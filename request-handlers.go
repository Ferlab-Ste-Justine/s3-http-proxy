package main

import (
  "context"
  "errors"
  "fmt"
  "io"
  "net/http"
  "regexp"
  "strings"
  "strconv"

  "github.com/gin-gonic/gin"
  "github.com/minio/minio-go/v7"
)

var rangeRegex *regexp.Regexp
func init() {
	rangeRegex = regexp.MustCompile(`bytes=(?P<start>\d+?)-(?P<end>\d+?)$`)
}

func parseRangeHeader(header string, dSize int64) (Range, error) {
	if header == "" {
		return Range{}, nil
	}

	if !rangeRegex.MatchString(header) {
		return Range{}, errors.New(fmt.Sprintf("Range header \"%s\" is incompatible with expected format of \"bytes=(\\d+?)-(\\d+?)\"", header))
	}

	r := Range{Enabled: true, Start: -1, End: -1}
	match := rangeRegex.FindStringSubmatch(header)
	if match[1] != "" {
		r.Start, _ = strconv.ParseInt(match[1], 10, 64)
	}
	if match[2] != "" {
		r.End, _ = strconv.ParseInt(match[2], 10, 64)
	}

	if r.Start == -1 {
		r.Start = 0
	}

	if r.End == -1 {
		r.End = dSize - 1
	}

	if r.Start >= dSize || r.End >= dSize {
		return r, errors.New(fmt.Sprintf("Range header \"%s\" falls outside what is permissible by a file with a size of %d", header, dSize))
	}

	if r.Start > r.End {
		return r, errors.New(fmt.Sprintf("Range header \"%s\" has start position after the end position", header))
	}

	return r, nil
}

type Handlers struct{
	GetS3File gin.HandlerFunc
	GetS3FileInfo gin.HandlerFunc
}

func GetHandlers(conf ConfigS3, cli *minio.Client) Handlers {
	
	getS3File := func(c *gin.Context) {
		fPath := c.Param("path")
		stat, statErr := cli.StatObject(context.Background(), conf.Bucket, fPath, minio.StatObjectOptions{})
		if statErr != nil {
			fmt.Println(fmt.Sprintf("Error occurred getting info on path %s on bucket %s: %s", fPath, conf.Bucket, statErr.Error()))			
			c.Data(
				http.StatusInternalServerError, 
				"application/text/plain", 
				[]byte(fmt.Sprintf("Error occurred getting info on path %s on bucket %s: %s", fPath, conf.Bucket, statErr.Error())),
			)
			return
		}

		downloadHandle, openErr := cli.GetObject(context.Background(), conf.Bucket, fPath, minio.GetObjectOptions{})
		if openErr != nil {
			fmt.Println(fmt.Sprintf("Error occurred getting download handle for file on path %s on bucket %s: %s", fPath, conf.Bucket, openErr.Error()))
			c.Data(
				http.StatusInternalServerError, 
				"application/text/plain", 
				[]byte(fmt.Sprintf("Error occurred getting download handle for file on path %s on bucket %s: %s", fPath, conf.Bucket, openErr.Error())),
			)
			return
		}
		defer downloadHandle.Close()

		reqRange, reqRangeErr := parseRangeHeader(c.GetHeader("Range"), stat.Size)
		if reqRangeErr != nil {
			c.Data(
				http.StatusBadRequest, 
				"application/text/plain", 
				[]byte(fmt.Sprintf("Error occurred while retrieving range information: %s", reqRangeErr.Error())),
			)
			return
		}

		c.Header("Accept-Ranges", "bytes")

		pathSegments := strings.Split(fPath, "/")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pathSegments[len(pathSegments)-1]))

		if !reqRange.Enabled {
			c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
			c.Status(http.StatusOK)
			_, copyErr := io.Copy(c.Writer, downloadHandle)
			if copyErr != nil {
				fmt.Println(fmt.Sprintf("Error occurred downloading file on path %s on bucket %s: %s", fPath, conf.Bucket, copyErr.Error()))
				return
			}
		} else {
			c.Header("Content-Length", fmt.Sprintf("%d", reqRange.End - reqRange.Start + 1))
			c.Status(http.StatusPartialContent)
			reader := RangeReader{
				ContentSize: stat.Size,
				Range:       reqRange,
				Handle:      downloadHandle,
			}
			_, copyErr := io.Copy(c.Writer, &reader)
			if copyErr != nil {
				fmt.Println(fmt.Sprintf("Error occurred downloading file on path %s on bucket %s: %s", fPath, conf.Bucket, copyErr.Error()))
				return
			}
		}

	}

	getS3FileInfo := func(c *gin.Context) {
		fPath := c.Param("path")
		stat, statErr := cli.StatObject(context.Background(), conf.Bucket, fPath, minio.StatObjectOptions{})
		if statErr != nil {
			fmt.Println(fmt.Sprintf("Error occurred getting info on path %s on bucket %s: %s", fPath, conf.Bucket, statErr.Error()))			
			c.Data(
				http.StatusInternalServerError, 
				"application/text/plain", 
				[]byte(fmt.Sprintf("Error occurred getting info on path %s on bucket %s: %s", fPath, conf.Bucket, statErr.Error())),
			)
			return
		}

		reqRange, reqRangeErr := parseRangeHeader(c.GetHeader("Range"), stat.Size)
		if reqRangeErr != nil {
			c.Data(
				http.StatusBadRequest, 
				"application/text/plain", 
				[]byte(fmt.Sprintf("Error occurred while retrieving range information: %s", reqRangeErr.Error())),
			)
			return
		}

		c.Header("Accept-Ranges", "bytes")

		pathSegments := strings.Split(fPath, "/")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pathSegments[len(pathSegments)-1]))
	
		if !reqRange.Enabled {
			c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
			c.Status(http.StatusOK)
		} else {
			c.Header("Content-Length", fmt.Sprintf("%d", reqRange.End - reqRange.Start + 1))
			c.Status(http.StatusPartialContent)
		}
	}

	return Handlers{
		GetS3File: getS3File,
		GetS3FileInfo: getS3FileInfo,
	}
}
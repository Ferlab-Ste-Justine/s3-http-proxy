package main

import (
  "context"
  "fmt"
  "io"
  "net/http"
  "strings"

  "github.com/gin-gonic/gin"
  "github.com/minio/minio-go/v7"
)


type Handlers struct{
	GetS3File gin.HandlerFunc
}

func GetHandlers(conf ConfigS3, cli *minio.Client) Handlers {
	
	getS3File := func(c *gin.Context) {
		fPath := c.Request.URL.Path
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

		pathSegments := strings.Split(fPath, "/")

		c.Header("Content-Type", "application/text/plain")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pathSegments[len(pathSegments)-1]))
		c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
		c.Status(http.StatusOK)
		_, copyErr := io.Copy(c.Writer, downloadHandle)
		if copyErr != nil {
			fmt.Println(fmt.Sprintf("Error occurred downloading file on path %s on bucket %s: %s", fPath, conf.Bucket, copyErr.Error()))
			return
		}
	}

	return Handlers{
		GetS3File: getS3File,
	}
}
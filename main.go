package main

import(
	"context"
  	"fmt"
  	"net/http"
  	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/gin-gonic/gin"
)

type Shutdown func() error

func Serve(conf Config, doneCh <-chan struct{}) <-chan error {
	var cli *minio.Client
	var server *http.Server
	errCh := make(chan error)

	shutdown := func() error {
		if server != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)	
			defer shutdownCancel()
			shutdownErr := server.Shutdown(shutdownCtx)
			if shutdownErr != nil && shutdownErr != http.ErrServerClosed {
				return shutdownErr
			}
		}

		return nil
	}

	go func() {
		defer func() {
			errCh <- shutdown()
			close(errCh)
		}()

		var err error
		cli, err = minio.New(conf.S3.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(conf.S3.AccessKey, conf.S3.SecretKey, ""),
			Secure: conf.S3.Tls,
			Region: conf.S3.Region,
		})
	
		if err != nil {
			errCh <- err
			return
		}
	
		if err != nil {
			errCh <- err
			return
		}

		if !conf.Server.DebugMode {
			gin.SetMode(gin.ReleaseMode)
		}

		accounts, accountsErr := getAccounts(conf)
		if accountsErr != nil {
			errCh <- accountsErr
			return	
		}

		router := gin.Default()
		server = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", conf.Server.Address, conf.Server.Port),
			Handler: router,
		}
	
		handlers := GetHandlers(conf.S3, cli)

		if len(accounts) > 0 {
			authorized := router.Group("/", gin.BasicAuth(accounts))
			authorized.GET("/*path", handlers.GetS3File)
		} else {
			router.GET("/*path", handlers.GetS3File)
		}

		serverDoneCh := make(chan error)
		go func() {
			defer close(serverDoneCh)
			if conf.Server.Tls.Certificate == "" {
				serverErr := server.ListenAndServe()
				if serverErr != nil && serverErr != http.ErrServerClosed {
					serverDoneCh <- serverErr
				}
			} else {
				serverErr := server.ListenAndServeTLS(conf.Server.Tls.Certificate, conf.Server.Tls.Key)
				if serverErr != nil && serverErr != http.ErrServerClosed {
					serverDoneCh <- serverErr
				}
			}
		}()

		select{
		case serverErr := <-serverDoneCh:
			if serverErr != nil {
				errCh <- serverErr
			}
		case <-doneCh:
		}
	}()

	return errCh
}

func main() {
	config, configErr := GetConfig()
	if configErr != nil {
		fmt.Println(configErr.Error())
		os.Exit(1)	
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	errCh := Serve(config, doneCh)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		fmt.Printf("Caught signal %s. Terminating.\n", sig.String())
		doneCh <- struct{}{}
		err := <-errCh
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		os.Exit(0)
	}()

	err := <-errCh
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)	
	}
}
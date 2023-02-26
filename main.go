package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/povsister/scp"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

func PushToS3() error {
	sugar.Infoln("File to push: ", os.Getenv("EXE_FILE"))
	sugar.Infoln("S3 Bucket is: ", os.Getenv("INPUT_S3_BUCKET"))
	sugar.Infoln("Release version is: ", os.Getenv("INPUT_RELEASE_VERSION"))

	os.Setenv("AWS_ACCESS_KEY_ID", os.Getenv("INPUT_AWS_ACCESS_KEY_ID"))
	os.Setenv("AWS_SECRET_ACCESS_KEY", os.Getenv("INPUT_AWS_SECRET_ACCESS_KEY"))
	os.Setenv("AWS_REGION", os.Getenv("INPUT_AWS_REGION"))

	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	svc := s3.New(sess)

	dirCnt, errB := os.ReadDir("./builds")
	if errB != nil {
		log.Fatal(errB)
	}

	for _, val := range dirCnt {
		if val.Name() == os.Getenv("EXE_FILE") || val.Name() == os.Getenv("VERSION_FILE") {
			sugar.Infof("%v/%v", "builds", val.Name())
			file := fmt.Sprintf("%v/%v", "builds", val.Name())

			buildfile, errF := os.Open(file)
			if errF != nil {
				log.Fatal(errF)
			}
			defer buildfile.Close()

			sugar.Infof("S3 path is %v/%v", os.Getenv("INPUT_RELEASE_VERSION"), val.Name())
			filekey := fmt.Sprintf("%v/%v",
				os.Getenv("INPUT_RELEASE_VERSION"),
				val.Name())

			input := &s3.PutObjectInput{
				Body:   buildfile,
				Bucket: aws.String(os.Getenv("INPUT_S3_BUCKET")),
				Key:    aws.String(filekey),
			}

			result, errPo := svc.PutObject(input)
			if errPo != nil {
				return fmt.Errorf("%w", errPo)
			}

			sugar.Infoln(result)
		}
	}

	s3BuildURL := fmt.Sprintf("s3://%v/%v/",
		os.Getenv("INPUT_S3_BUCKET"),
		os.Getenv("INPUT_RELEASE_VERSION"))
	sugar.Infof("The s3 url for the build is: %v", s3BuildURL)
	// sugar.Infoln(fmt.Sprintln(`::set-output name=s3_build_url::`, s3_build_url))
	// echo "time=$time" >> $GITHUB_OUTPUT

	cmd := "echo"
	arg1 := fmt.Sprintf("%v%v", "'s3_build_url=", s3BuildURL)
	arg2 := ">> $GITHUB_OUTPUT'"
	exeCmd := exec.Command(cmd, arg1, arg2)
	sugar.Infoln("Running Command: ", cmd, arg1, arg2)

	if errR := exeCmd.Run(); errR != nil {
		sugar.Errorln(errR)

		return fmt.Errorf("%w", errR)
	}

	return nil
}

func DeployBuildToEC2() error {
	sugar.Infoln("Setting the user session.")

	sshConf := scp.NewSSHConfigFromPassword(os.Getenv("INPUT_EC2_USER"), os.Getenv("INPUT_EC2_PASS"))

	scpClient, err := scp.NewClient(os.Getenv("INPUT_EC2_IP"), sshConf, &scp.ClientOption{})

	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer scpClient.Close()

	dirCnt, err2 := os.ReadDir("./builds")
	if err2 != nil {
		sugar.Errorln(err2)

		return fmt.Errorf("%w", err2)
	}

	for _, val := range dirCnt {
		locfile := fmt.Sprintf("%v/%v", "builds", val.Name())
		remfile := fmt.Sprintf("%v/%v", os.Getenv("INPUT_EC2_PATH"), val.Name())
		sugar.Infoln("Transferring the file", locfile, "to", remfile)

		if val.Name() == os.Getenv("EXE_FILE") || val.Name() == os.Getenv("VERSION_FILE") {
			err = scpClient.CopyFileToRemote(locfile, remfile, &scp.FileTransferOption{Perm: 0o755, PreserveProp: false})
		}

		if err != nil {
			sugar.Errorln(err)

			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

func createDeployMetaFile() (string, error) {
	currtime := time.Now()

	sugar.Infoln("Deployed Filename: ", os.Getenv("EXE_FILE"))
	sugar.Infoln("Deployed Version: ", os.Getenv("INPUT_RELEASE_VERSION"))
	sugar.Infoln("Deployed Timestamp: ", currtime.Round(0))
	sugar.Infoln("Deployed By: ", os.Getenv("GITHUB_ACTOR"))

	data := []byte(fmt.Sprintf(
		"Deployed Filename: %v\nDeployed Version: %v\nDeployed Timestamp: %v\nDeployed By:%v\n",
		os.Getenv("EXE_FILE"),
		os.Getenv("INPUT_RELEASE_VERSION"),
		currtime.Round(0),
		os.Getenv("GITHUB_ACTOR")))

	metafile := fmt.Sprintf("%v/%v", "builds",
		os.Getenv("VERSION_FILE"))

	metaFileName, err := os.Create(metafile)
	if err != nil {
		sugar.Errorln(err)

		return "", fmt.Errorf("%w", err)
	}
	defer metaFileName.Close()

	_, err1 := metaFileName.Write(data)
	if err1 != nil {
		sugar.Errorln(err1)

		return "", fmt.Errorf("%w", err1)
	}

	return metafile, nil
}

func makeDir() error {
	if _, err := os.Stat("."); errors.Is(err, os.ErrNotExist) {
		errM := os.Mkdir("builds", 0o755)
		if errM != nil {
			return fmt.Errorf("%w", errM)
		}
	}

	return nil
}

func cleanup() error {
	sugar.Info("Cleaning local...")
	if err := os.RemoveAll("./builds"); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func init() {
	logger, err1 := zap.NewProduction()
	if err1 != nil {
		sugar.Errorln(err1)
	}

	sugar = logger.Sugar()
}

func main() {
	sugar.Infoln("Getting the values.")
	sugar.Infoln("Executable Name: ",
		os.Getenv("INPUT_EXECUTABLE_NAME"))
	sugar.Infoln("Go os is set to ",
		os.Getenv("INPUT_GOOS"))
	sugar.Infoln("Go Arch is set to ",
		os.Getenv("INPUT_GOARCH"))
	sugar.Infoln("AWS Region is set to ",
		os.Getenv("INPUT_AWS_REGION"))
	sugar.Infoln("S3 bucket is set to: ",
		os.Getenv("INPUT_S3_BUCKET"))
	sugar.Infoln("Release version is set to: ",
		os.Getenv("INPUT_RELEASE_VERSION"))

	if err := os.Setenv("RELEASE_VERSION",
		strings.ReplaceAll(os.Getenv("INPUT_RELEASE_VERSION"),
			".", "")); err != nil {
		sugar.Errorln(err)
	}

	if os.Getenv("INPUT_GOOS") == "windows" {
		os.Setenv("EXE_FILE", fmt.Sprintf("%v-%v.exe",
			os.Getenv("INPUT_EXECUTABLE_NAME"),
			os.Getenv("RELEASE_VERSION")))
	} else {
		os.Setenv("EXE_FILE", fmt.Sprintf("%v-%v",
			os.Getenv("INPUT_EXECUTABLE_NAME"),
			os.Getenv("RELEASE_VERSION")))
	}

	if err := os.Setenv("GOOS",
		os.Getenv("INPUT_GOOS")); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("GOARCH",
		os.Getenv("INPUT_GOARCH")); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("GOHOSTOS", "linux"); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("GOHOSTARCH", "amd64"); err != nil {
		log.Fatal(err)
	}

	sugar.Infoln("GOOS: ", os.Getenv("GOOS"),
		",GOARCH: ", os.Getenv("GOARCH"),
		",GOHOSTOS: ", os.Getenv("GOHOSTOS"),
		",GOHOSTARCH: ", os.Getenv("GOHOSTARCH"))

	verMetaFile := fmt.Sprintf("%v-%v.txt", "meta", os.Getenv("RELEASE_VERSION"))

	if err := os.Setenv("VERSION_FILE",
		verMetaFile); err != nil {
		sugar.Errorln(err)
	}

	sugar.Infoln("Cleaning up old build directory.")

	if err := cleanup(); err != nil {
		sugar.Errorln("Builds directory does not exist.")
	}

	sugar.Infoln("Creating builds directory.\n")

	if err := makeDir(); err != nil {
		sugar.Errorln(err)
	}

	cmd := "go"
	arg1 := "build"
	arg2 := "-o"
	arg3 := fmt.Sprintf("%v/%v", "builds", os.Getenv("EXE_FILE"))
	exe := exec.Command(cmd, arg1, arg2, arg3)
	sugar.Infoln("Running Command: ", cmd, arg1, arg2, arg3)

	if err := exe.Run(); err != nil {
		sugar.Errorln(err)
	}

	_, err := createDeployMetaFile()
	if err != nil {
		sugar.Errorln(err)
	}

	if os.Getenv("INPUT_PUSH_TO_EC2") == "true" && os.Getenv("INPUT_PUSH_TO_S3") == "true" {
		sugar.Infoln("PUSH_TO_S3 is set to true, Pushing build to s3.")

		if errS := PushToS3(); errS != nil {
			sugar.Errorln(errS)
		}

		sugar.Infoln("PUSH_TO_EC2 is set to true Pushing build to ec2.")

		if errE := DeployBuildToEC2(); errE != nil {
			sugar.Errorln(errE)
		}

		if errC := cleanup(); errC != nil {
			sugar.Errorln(errC)
		}
	} else if os.Getenv("INPUT_PUSH_TO_S3") == "true" {
		sugar.Infoln("PUSH_TO_S3 is set to true, Pushing build to s3.")

		if errS := PushToS3(); errS != nil {
			sugar.Errorln(errS)
		}

		if errC := cleanup(); errC != nil {
			sugar.Errorln(errC)
		}
	} else if os.Getenv("INPUT_PUSH_TO_EC2") == "true" {
		sugar.Infoln("PUSH_TO_EC2 is set to true Pushing build to ec2.")

		if errE := DeployBuildToEC2(); errE != nil {
			sugar.Errorln(errE)
		}

		if errC := cleanup(); errC != nil {
			sugar.Errorln(errC)
		}
	} else {
		sugar.Errorln("No input to push to s3 or ec2, exiting")
		os.Exit(1)
	}

	sugar.Infoln("GITHUB_OUTPUT", os.Getenv("GITHUB_OUTPUT"))
	sugar.Infoln("Process completed successfully :)")
}

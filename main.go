package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/povsister/scp"
)

func PushToS3() error {

	fmt.Printf("File to push: %v\n", os.Getenv("EXE_FILE"))
	fmt.Printf("S3 Bucket is: %v\n", os.Getenv("INPUT_S3_BUCKET"))
	fmt.Printf("Release version is: %v\n", os.Getenv("INPUT_RELEASE_VERSION"))

	os.Setenv("AWS_ACCESS_KEY_ID", os.Getenv("INPUT_AWS_ACCESS_KEY_ID"))
	os.Setenv("AWS_SECRET_ACCESS_KEY", os.Getenv("INPUT_AWS_SECRET_ACCESS_KEY"))
	os.Setenv("AWS_REGION", os.Getenv("INPUT_AWS_REGION"))

	sess, err := session.NewSession()

	if err != nil {
		return err
	}

	svc := s3.New(sess)

	dirCnt, err := ioutil.ReadDir("./builds")
	if err != nil {
		log.Fatal(err)
	}

	for _, val := range dirCnt {
		if val.Name() == os.Getenv("EXE_FILE") || val.Name() == os.Getenv("VERSION_FILE") {
			fmt.Printf("%v/%v\n", "builds", val.Name())
			file := fmt.Sprintf("%v/%v", "builds", val.Name())

			buildfile, err := os.Open(file)
			if err != nil {
				log.Fatal(err)
			}
			defer buildfile.Close()

			fmt.Printf("S3 path is %v/%v\n", os.Getenv("INPUT_RELEASE_VERSION"), val.Name())
			filekey := fmt.Sprintf("%v/%v",
				os.Getenv("INPUT_RELEASE_VERSION"),
				val.Name())

			input := &s3.PutObjectInput{
				Body:   buildfile,
				Bucket: aws.String(os.Getenv("INPUT_S3_BUCKET")),
				Key:    aws.String(filekey),
			}

			result, err := svc.PutObject(input)
			if err != nil {
				return err
			}
			fmt.Sprintln(result)
		}
	}

	s3_build_url := fmt.Sprintf("s3://%v/%v/",
		os.Getenv("INPUT_S3_BUCKET"),
		os.Getenv("INPUT_RELEASE_VERSION"))

	fmt.Printf("The s3 url for the build is: %v\n", s3_build_url)
	fmt.Printf("%v\n", fmt.Sprintf(`::set-output name=s3_build_url::%v`, s3_build_url))

	return nil
}

func DeployBuildToEC2() error {

	fmt.Printf("Setting the user session.\n")
	sshConf := scp.NewSSHConfigFromPassword(
		os.Getenv("INPUT_EC2_USER"),
		os.Getenv("INPUT_EC2_PASS"))

	scpClient, err := scp.NewClient(os.Getenv("INPUT_EC2_IP"), sshConf, &scp.ClientOption{})
	if err != nil {
		return err
	}
	defer scpClient.Close()

	dirCnt, err2 := ioutil.ReadDir("./builds")
	if err2 != nil {
		log.Fatal(err2)
	}

	for _, val := range dirCnt {
		locfile := fmt.Sprintf("%v/%v", "builds", val.Name())
		remfile := fmt.Sprintf("%v/%v", os.Getenv("INPUT_EC2_PATH"), val.Name())
		fmt.Printf("Transferring the file %v to %v\n", locfile, remfile)
		if val.Name() == os.Getenv("EXE_FILE") || val.Name() == os.Getenv("VERSION_FILE") {
			err = scpClient.CopyFileToRemote(
				locfile,
				remfile,
				&scp.FileTransferOption{Perm: 0755, PreserveProp: false})
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func createDeployMetaFile() (string, error) {
	currtime := time.Now()
	fmt.Printf("Deployed Filename: %v\n", os.Getenv("INPUT_EXECUTABLE_NAME"))
	fmt.Printf("Deployed Version: %v\n", os.Getenv("INPUT_RELEASE_VERSION"))
	fmt.Printf("Deployed Timestamp: %v\n", currtime.Round(0))

	data := []byte(fmt.Sprintf(
		"Deployed Filename: %v\nDeployed Version: %v\nDeployed Timestamp: %v\n",
		os.Getenv("INPUT_EXECUTABLE_NAME"),
		os.Getenv("INPUT_RELEASE_VERSION"),
		currtime.Round(0)))

	metafile := fmt.Sprintf("%v/%v", "builds",
		os.Getenv("VERSION_FILE"))
	f, err := os.Create(metafile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err1 := f.Write(data)
	if err1 != nil {
		log.Fatal(err1)
	}

	return metafile, nil
}

func makeDir() error {
	err := os.Mkdir("builds", 0755)
	if err != nil {
		return err
	}

	return nil
}

func cleanup() error {
	if err := os.RemoveAll("./builds"); err != nil {
		return err
	}
	return nil
}

func main() {
	fmt.Println("Getting the values.")
	fmt.Printf("Executable Name: %v\n", os.Getenv("INPUT_EXECUTABLE_NAME"))
	fmt.Printf("Go os is set to %v\n", os.Getenv("INPUT_GOOS"))
	fmt.Printf("Go Arch is set to %v\n", os.Getenv("INPUT_GOARCH"))
	fmt.Printf("AWS Region is set to %v\n", os.Getenv("INPUT_AWS_REGION"))
	fmt.Printf("S3 bucket is set to: %v\n", os.Getenv("INPUT_S3_BUCKET"))
	fmt.Printf("Release version is set to: %v\n", os.Getenv("INPUT_RELEASE_VERSION"))

	if err := os.Setenv("RELEASE_VERSION", strings.Replace(os.Getenv("INPUT_RELEASE_VERSION"), ".", "", -1)); err != nil {
		log.Fatal(err)
	}

	if os.Getenv("INPUT_GOOS") == "windows" {
		os.Setenv("EXE_FILE", fmt.Sprintf("%v-%v.exe", os.Getenv("INPUT_EXECUTABLE_NAME"), os.Getenv("RELEASE_VERSION")))
	} else {
		os.Setenv("EXE_FILE", fmt.Sprintf("%v-%v", os.Getenv("INPUT_EXECUTABLE_NAME"), os.Getenv("RELEASE_VERSION")))
	}

	if err := os.Setenv("GOOS", os.Getenv("INPUT_GOOS")); err != nil {
		log.Fatal(err)
	}
	if err := os.Setenv("GOARCH", os.Getenv("INPUT_GOARCH")); err != nil {
		log.Fatal(err)
	}

	ver_meta_file := fmt.Sprintf("%v-%v.txt", "VersionMeta", os.Getenv("RELEASE_VERSION"))
	if err := os.Setenv("VERSION_FILE", ver_meta_file); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Creating builds directory.\n")
	if err := makeDir(); err != nil {
		log.Fatal(err)
	}

	cmd := "go"
	arg1 := "build"
	arg2 := "-o"
	arg3 := fmt.Sprintf("%v/%v", "builds", os.Getenv("EXE_FILE"))
	exe := exec.Command(cmd, arg1, arg2, arg3)
	fmt.Printf("Running Command: %v %v %v %v\n", cmd, arg1, arg2, arg3)

	if err := exe.Run(); err != nil {
		log.Fatal(err)
	}

	_, err := createDeployMetaFile()
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("INPUT_PUSH_TO_EC2") == "true" && os.Getenv("INPUT_PUSH_TO_S3") == "true" {
		fmt.Println("PUSH_TO_S3 is set to true, Pushing build to s3.")
		if err := PushToS3(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("PUSH_TO_EC2 is set to true Pushing build to ec2.")
		if err := DeployBuildToEC2(); err != nil {
			log.Fatal(err)
		}
		if err := cleanup(); err != nil {
			log.Fatal(err)
		}
	} else if os.Getenv("INPUT_PUSH_TO_S3") == "true" {
		fmt.Println("PUSH_TO_S3 is set to true, Pushing build to s3.")
		if err := PushToS3(); err != nil {
			log.Fatal(err)
		}
		if err := cleanup(); err != nil {
			log.Fatal(err)
		}
	} else if os.Getenv("INPUT_PUSH_TO_EC2") == "true" {
		fmt.Println("PUSH_TO_EC2 is set to true Pushing build to ec2.")
		if err := DeployBuildToEC2(); err != nil {
			log.Fatal(err)
		}
		if err := cleanup(); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("No input to push to s3 or ec2, exiting")
	}

}

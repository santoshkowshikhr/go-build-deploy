# go-build-deploy

This github action will build binary and push the binary to ec2 or s3.

[![build](https://github.com/santoshkowshikhr/go-build-deploy/actions/workflows/build.yml/badge.svg?branch=test-github-actions)](https://github.com/santoshkowshikhr/go-build-deploy/actions/workflows/build.yml)

[![GitHub license](https://img.shields.io/badge/license-GNU%20GPL-blue)](https://github.com/santoshkowshikhr/go-build-deploy/blob/main/LICENSE)

This will help teams automate build and deploy the executable to ec2 or s3.

## Usage:
```
steps:
- name: Start build deploy
  uses: santoshkowshikhr/go-build-deploy@v1.1.0
  with:
    executable_name: go-executable
    goos: linux
    goarch: amd64
    aws_region: us-east-1
    aws_access_key_id: ${{ secrets.AWS_ACCESS_KEY_ID }
    aws_secret_access_key: ${{ AWS_SECRET_ACCESS_KEY }}
    s3_bucket: my-go-build
    release_version: v1.1.1
    push_to_s3: true
    push_to_ec2: true
    ec2_user: ubuntu
    ec2_pass: ${{ secrets.password }}
    ec2_ip: ${{ secrets.ec2_ip }}
    ec2_path: /data
```

### Inputs:
| Name | Description | Default |Required | Comments |
| - | - | - | - | - |
| **`executable_name`** | This is the executable name that will be stored on to ec2 and s3. | Github repo name | ✔ | |
| **`goos`** | This is the os name for which the executable needs to be built. | linux | | |
| **`goarch`** | This is the architecture of os the executable needs to be built. | amd64 | | |
| **`aws_region`** | AWS region to set for the account. | ***us-east-1*** | | |
| **`aws_access_key_id`** | AWS Access Key ID for the user with s3 access. ***Store it in github secrets for security reasons***. | - | ✔ | Required if push_to_s3 is **true**|
| **`aws_secret_access_key`** | AWS Access Key ID for the user with s3 access. ***Store it in secrets for security reasons***. | - | ✔ | Required if push_to_s3 is **true** |
| **`s3_bucket`** | The s3 bucket to push the file to. | | ✔ | Required if push_to_s3 is **true** |
| **`release_version`** | The version tag to append to filename. Use the github actions to fetch the version tag of current event(**GITHUB_REF#refs/*/**). | ***v0.0.0*** | | |
| **`push_to_s3`** | Set this to true if the executable needs to be pushed to s3. | ***false*** | | |
| **`push_to_ec2`** | Set this to true if the executable needs to be pushed to ec2. | ***false*** | | |
| **`ec2_user`** | The user of the ec2 user with valid password. | | ✔ | Required if push_to_ec2 is **true** |
| **`ec2_pass`** | The password to the ec2 user. ***Store the password in github secrets for security reasons***. | | ✔ | Required if push_to_ec2 is **true** |
| **`ec2_ip`** | Public IP Address of the ec2 instance. ***Store the public IP in github secrets for security reasons***. | | ✔ | Required if push_to_ec2 is **true** |
| **`ec2_path`** | The path on the ec2 to push the executable to.| | ✔ | Required if push_to_ec2 is **true** |



***`Note:`***
- This currently supports only username and password, future versions may include ssh keys to login.
- The remote directory must already exist, this action will not create folder.
- Refer to this for supposrted [list](https://github.com/santoshkowshikhr/go-build-deploy/blob/main/go_dist_list.txt) of os and architecture.


### Outputs:
| Name | Description |
| --- | --- |
| **`s3_url`** | The s3 url where the build was pushed. |

## License
The scripts and documentation in this project are released under the [GNU GPL](https://github.com/santoshkowshikhr/go-build-deploy/blob/main/LICENSE).

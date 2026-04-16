package video

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	vod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vod/v20180717"
)

const (
	// QCloud default region for VOD
	QCloudVODDefaultRegion = "ap-guangzhou"
)

// QCloudVideoCensor is the client for Tencent Cloud video content moderation
type QCloudVideoCensor struct {
	SecretID  string
	SecretKey string
	Region    string
	Client    *vod.Client
}

// NewQCloudVideoCensor creates a new Tencent Cloud video moderation client
func NewQCloudVideoCensor(secretID, secretKey, region string) (*QCloudVideoCensor, error) {
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("secretID and secretKey cannot be empty")
	}

	if region == "" {
		region = QCloudVODDefaultRegion
	}

	// Create credential
	credential := common.NewCredential(secretID, secretKey)

	// Create client profile
	cpf := profile.NewClientProfile()

	// Create VOD client
	client, err := vod.NewClient(credential, region, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent Cloud VOD client: %w", err)
	}

	return &QCloudVideoCensor{
		SecretID:  secretID,
		SecretKey: secretKey,
		Region:    region,
		Client:    client,
	}, nil
}

// SubmitCensorVideo submits a video moderation task
func (c *QCloudVideoCensor) SubmitCensorVideo(videoURL string) (string, error) {
	if videoURL == "" {
		return "", fmt.Errorf("videoURL cannot be empty")
	}

	// For Tencent Cloud, video moderation is handled through the VOD service
	// This is a placeholder implementation that returns a generated task ID
	// In production, you would need to implement the actual API call based on Tencent Cloud's documentation
	taskID := fmt.Sprintf("qcloud-video-%d", int64(len(videoURL)))
	return taskID, nil
}

// SubmitCensorVideoByFileID submits a video moderation task using file ID
func (c *QCloudVideoCensor) SubmitCensorVideoByFileID(fileID string) (string, error) {
	if fileID == "" {
		return "", fmt.Errorf("fileID cannot be empty")
	}

	// Create request for video content review
	request := vod.NewProcessMediaRequest()
	request.FileId = common.StringPtr(fileID)

	// Send request
	response, err := c.Client.ProcessMedia(request)
	if err != nil {
		return "", fmt.Errorf("failed to call Tencent Cloud video moderation API: %w", err)
	}

	if response.Response == nil || response.Response.TaskId == nil {
		return "", fmt.Errorf("empty response from Tencent Cloud")
	}

	return *response.Response.TaskId, nil
}

// GetCensorResult retrieves the video moderation result
func (c *QCloudVideoCensor) GetCensorResult(taskID string) (interface{}, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	// Create request to get task info
	request := vod.NewDescribeTaskDetailRequest()
	request.TaskId = common.StringPtr(taskID)

	// Send request
	response, err := c.Client.DescribeTaskDetail(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get task result: %w", err)
	}

	if response.Response == nil {
		return nil, fmt.Errorf("empty response from Tencent Cloud")
	}

	return response.Response, nil
}

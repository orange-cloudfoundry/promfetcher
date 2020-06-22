package models

type AppEndpoint struct {
	GUID     string `gorm:"primary_key"`
	AppGUID  string
	Endpoint string
}

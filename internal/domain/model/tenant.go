package model

import "time"

type Tenant struct {
	ID           string
	DisplayName  string
	Province     string
	City         string
	District     string
	Address      string
	ContactName  string
	ContactPhone string
	Remark       string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TenantListItem struct {
	Tenant             *Tenant
	TenantAdminUserID  string
	TenantAdminAccount string
	TenantAdminName    string
	TenantAdminPhone   string
}

type TenantCreateInput struct {
	TenantID             string
	DisplayName          string
	Province             string
	City                 string
	District             string
	Address              string
	ContactName          string
	ContactPhone         string
	Remark               string
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	TenantAdminUserID    string
	TenantAdminAccount   string
	TenantAdminName      string
	TenantAdminPhone     string
	TenantAdminPassword  string
	TenantAdminCreatedAt time.Time
}

type TenantCreateOutput struct {
	TenantID           string
	TenantAdminUserID  string
	TenantAdminAccount string
	TenantAdminName    string
	Status             string
	CreatedAt          time.Time
}

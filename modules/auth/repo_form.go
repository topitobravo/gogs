// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/i18n"

	"github.com/topitobravo/gogs/modules/middleware/binding"
)

// _______________________________________    _________.______________________ _______________.___.
// \______   \_   _____/\______   \_____  \  /   _____/|   \__    ___/\_____  \\______   \__  |   |
//  |       _/|    __)_  |     ___//   |   \ \_____  \ |   | |    |    /   |   \|       _//   |   |
//  |    |   \|        \ |    |   /    |    \/        \|   | |    |   /    |    \    |   \\____   |
//  |____|_  /_______  / |____|   \_______  /_______  /|___| |____|   \_______  /____|_  // ______|
//         \/        \/                   \/        \/                        \/       \/ \/

type CreateRepoForm struct {
	Uid         int64  `form:"uid" binding:"Required"`
	RepoName    string `form:"repo_name" binding:"Required;AlphaDashDot;MaxSize(100)"`
	Private     bool   `form:"private"`
	Description string `form:"desc" binding:"MaxSize(255)"`
	Gitignore   string `form:"gitignore"`
	License     string `form:"license"`
	InitReadme  bool   `form:"init_readme"`
}

func (f *CreateRepoForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

type MigrateRepoForm struct {
	HttpsUrl     string `form:"url" binding:"Required;Url"`
	AuthUserName string `form:"auth_username"`
	AuthPasswd   string `form:"auth_password"`
	Uid          int64  `form:"uid" binding:"Required"`
	RepoName     string `form:"repo_name" binding:"Required;AlphaDashDot;MaxSize(100)"`
	Mirror       bool   `form:"mirror"`
	Private      bool   `form:"private"`
	Description  string `form:"desc" binding:"MaxSize(255)"`
}

func (f *MigrateRepoForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

type RepoSettingForm struct {
	RepoName    string `form:"repo_name" binding:"Required;AlphaDashDot;MaxSize(100)"`
	Description string `form:"desc" binding:"MaxSize(255)"`
	Website     string `form:"site" binding:"Url;MaxSize(100)"`
	Branch      string `form:"branch"`
	Interval    int    `form:"interval"`
	Private     bool   `form:"private"`
	GoGet       bool   `form:"goget"`
}

func (f *RepoSettingForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

//  __      __      ___.   .__    .__            __
// /  \    /  \ ____\_ |__ |  |__ |  |__   ____ |  | __
// \   \/\/   // __ \| __ \|  |  \|  |  \ /  _ \|  |/ /
//  \        /\  ___/| \_\ \   Y  \   Y  (  <_> )    <
//   \__/\  /  \___  >___  /___|  /___|  /\____/|__|_ \
//        \/       \/    \/     \/     \/            \/

type NewWebhookForm struct {
	HookTaskType string `form:"hook_type" binding:"Required"`
	PayloadUrl   string `form:"payload_url" binding:"Required;Url"`
	ContentType  string `form:"content_type" binding:"Required"`
	Secret       string `form:"secret"`
	PushOnly     bool   `form:"push_only"`
	Active       bool   `form:"active"`
}

func (f *NewWebhookForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

type NewSlackHookForm struct {
	HookTaskType string `form:"hook_type" binding:"Required"`
	Domain       string `form:"domain" binding:"Required`
	Token        string `form:"token" binding:"Required"`
	Channel      string `form:"channel" binding:"Required"`
	PushOnly     bool   `form:"push_only"`
	Active       bool   `form:"active"`
}

func (f *NewSlackHookForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

// .___
// |   | ______ ________ __   ____
// |   |/  ___//  ___/  |  \_/ __ \
// |   |\___ \ \___ \|  |  /\  ___/
// |___/____  >____  >____/  \___  >
//          \/     \/            \/

type CreateIssueForm struct {
	IssueName   string `form:"title" binding:"Required;MaxSize(255)"`
	MilestoneId int64  `form:"milestoneid"`
	AssigneeId  int64  `form:"assigneeid"`
	Labels      string `form:"labels"`
	Content     string `form:"content"`
}

func (f *CreateIssueForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

//    _____  .__.__                   __
//   /     \ |__|  |   ____   _______/  |_  ____   ____   ____
//  /  \ /  \|  |  | _/ __ \ /  ___/\   __\/  _ \ /    \_/ __ \
// /    Y    \  |  |_\  ___/ \___ \  |  | (  <_> )   |  \  ___/
// \____|__  /__|____/\___  >____  > |__|  \____/|___|  /\___  >
//         \/             \/     \/                   \/     \/

type CreateMilestoneForm struct {
	Title    string `form:"title" binding:"Required;MaxSize(50)"`
	Content  string `form:"content"`
	Deadline string `form:"due_date"`
}

func (f *CreateMilestoneForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

// .____          ___.          .__
// |    |   _____ \_ |__   ____ |  |
// |    |   \__  \ | __ \_/ __ \|  |
// |    |___ / __ \| \_\ \  ___/|  |__
// |_______ (____  /___  /\___  >____/
//         \/    \/    \/     \/

type CreateLabelForm struct {
	Title string `form:"title" binding:"Required;MaxSize(50)"`
	Color string `form:"color" binding:"Required;Size(7)"`
}

func (f *CreateLabelForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

// __________       .__
// \______   \ ____ |  |   ____ _____    ______ ____
//  |       _// __ \|  | _/ __ \\__  \  /  ___// __ \
//  |    |   \  ___/|  |_\  ___/ / __ \_\___ \\  ___/
//  |____|_  /\___  >____/\___  >____  /____  >\___  >
//         \/     \/          \/     \/     \/     \/

type NewReleaseForm struct {
	TagName    string `form:"tag_name" binding:"Required"`
	Target     string `form:"tag_target" binding:"Required"`
	Title      string `form:"title" binding:"Required"`
	Content    string `form:"content" binding:"Required"`
	Draft      string `form:"draft"`
	Prerelease bool   `form:"prerelease"`
}

func (f *NewReleaseForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

type EditReleaseForm struct {
	Target     string `form:"tag_target" binding:"Required"`
	Title      string `form:"title" binding:"Required"`
	Content    string `form:"content" binding:"Required"`
	Draft      string `form:"draft"`
	Prerelease bool   `form:"prerelease"`
}

func (f *EditReleaseForm) Validate(ctx *macaron.Context, errs *binding.Errors, l i18n.Locale) {
	validate(errs, ctx.Data, f, l)
}

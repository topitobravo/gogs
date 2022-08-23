// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package v1

import (
	"github.com/Unknwon/com"

	"github.com/topitobravo/gogs/models"
	"github.com/topitobravo/gogs/modules/middleware"
)

type user struct {
	UserName   string `json:"username"`
	AvatarLink string `json:"avatar"`
}

func SearchUsers(ctx *middleware.Context) {
	opt := models.SearchOption{
		Keyword: ctx.Query("q"),
		Limit:   com.StrTo(ctx.Query("limit")).MustInt(),
	}
	if opt.Limit == 0 {
		opt.Limit = 10
	}

	us, err := models.SearchUserByName(opt)
	if err != nil {
		ctx.JSON(500, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	results := make([]*user, len(us))
	for i := range us {
		results[i] = &user{
			UserName:   us[i].Name,
			AvatarLink: us[i].AvatarLink(),
		}
	}

	ctx.Render.JSON(200, map[string]interface{}{
		"ok":   true,
		"data": results,
	})
}

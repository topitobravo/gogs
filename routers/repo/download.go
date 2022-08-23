// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"io"
	"path"

	"github.com/topitobravo/gogs/modules/base"
	"github.com/topitobravo/gogs/modules/middleware"
)

func SingleDownload(ctx *middleware.Context) {
	treename := ctx.Params("*")

	blob, err := ctx.Repo.Commit.GetBlobByPath(treename)
	if err != nil {
		ctx.Handle(500, "GetBlobByPath", err)
		return
	}

	dataRc, err := blob.Data()
	if err != nil {
		ctx.Handle(500, "repo.SingleDownload(Data)", err)
		return
	}

	buf := make([]byte, 1024)
	n, _ := dataRc.Read(buf)
	if n > 0 {
		buf = buf[:n]
	}

	contentType, isTextFile := base.IsTextFile(buf)
	_, isImageFile := base.IsImageFile(buf)
	ctx.Resp.Header().Set("Content-Type", contentType)
	if !isTextFile && !isImageFile {
		ctx.Resp.Header().Set("Content-Disposition", "attachment; filename="+path.Base(treename))
		ctx.Resp.Header().Set("Content-Transfer-Encoding", "binary")
	}
	ctx.Resp.Write(buf)
	io.Copy(ctx.Resp, dataRc)
}

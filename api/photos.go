package api

import (
	"github.com/zenazn/goji/web"
	"net/http"
	"strconv"
	"strings"
)

func getPhotoOr404(c web.C, w http.ResponseWriter, r *http.Request) (*Photo, bool) {
	photoID, err := strconv.ParseInt(c.URLParams["id"], 10, 0)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}
	photo, exists, err := photoMgr.Get(photoID)
	if err != nil {
		serverError(w, err)
		return photo, false
	}
	if !exists {
		http.NotFound(w, r)
		return photo, false
	}
	return photo, true
}

func deletePhoto(c web.C, w http.ResponseWriter, r *http.Request) {

	user, ok := getUserOr401(w, r)
	if !ok {
		return
	}

	photo, ok := getPhotoOr404(c, w, r)
	if !ok {
		return
	}

	if !photo.CanDelete(user) {
		http.Error(w, "You can't delete this photo", http.StatusForbidden)
		return
	}
	if err := photoMgr.Delete(photo); err != nil {
		serverError(w, err)
		return
	}

	sendMessage(&SocketMessage{user.Name, "", photo.ID, "photo_deleted"})
	w.WriteHeader(http.StatusNoContent)
}

func photoDetail(c web.C, w http.ResponseWriter, r *http.Request) {

	user, err := getCurrentUser(r)
	if err != nil {
		serverError(w, err)
		return
	}

	photoID, err := strconv.ParseInt(c.URLParams["id"], 10, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	photo, exists, err := photoMgr.GetDetail(photoID, user)
	if err != nil {
		serverError(w, err)
		return
	}
	if !exists {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, photo, http.StatusOK)
}

func getPhotoToEdit(c web.C, w http.ResponseWriter, r *http.Request) (*Photo, bool) {
	user, ok := getUserOr401(w, r)
	if !ok {
		return nil, false
	}

	photo, ok := getPhotoOr404(c, w, r)
	if !ok {
		return photo, false
	}

	if !photo.CanEdit(user) {
		http.Error(w, "You can't edit this photo", http.StatusForbidden)
		return photo, false
	}
	return photo, true
}

func editPhotoTitle(c web.C, w http.ResponseWriter, r *http.Request) {

	photo, ok := getPhotoToEdit(c, w, r)

	if !ok {
		return
	}

	s := &struct {
		Title string `json:"title"`
	}{}

	if err := decodeJSON(r, s); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	photo.Title = s.Title

	validator := getPhotoValidator(photo)

	if result, err := formHandler.Validate(validator); err != nil || !result.OK {
		if err != nil {
			serverError(w, err)
			return
		}
		result.Write(w)
		return
	}

	if err := photoMgr.Update(photo); err != nil {
		serverError(w, err)
		return
	}
	if user, err := getCurrentUser(r); err == nil {
		sendMessage(&SocketMessage{user.Name, "", photo.ID, "photo_updated"})
	}
	w.WriteHeader(http.StatusNoContent)
}

func editPhotoTags(c web.C, w http.ResponseWriter, r *http.Request) {

	photo, ok := getPhotoToEdit(c, w, r)

	if !ok {
		return
	}

	s := &struct {
		Tags []string `json:"tags"`
	}{}

	if err := decodeJSON(r, s); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	photo.Tags = s.Tags

	if err := photoMgr.UpdateTags(photo); err != nil {
		serverError(w, err)
		return
	}
	if user, err := getCurrentUser(r); err == nil {
		sendMessage(&SocketMessage{user.Name, "", photo.ID, "photo_updated"})
	}
	w.WriteHeader(http.StatusNoContent)
}

func upload(w http.ResponseWriter, r *http.Request) {

	user, ok := getUserOr401(w, r)
	if !ok {
		return
	}

	title := r.FormValue("title")
	taglist := r.FormValue("taglist")
	tags := strings.Split(taglist, " ")

	src, hdr, err := r.FormFile("photo")
	if err != nil {
		if err == http.ErrMissingFile || err == http.ErrNotMultipart {
			http.Error(w, "No image was posted", http.StatusBadRequest)
			return
		}
		serverError(w, err)
		return
	}
	defer src.Close()

	contentType := hdr.Header["Content-Type"][0]

	filename, err := imageProcessor.Process(src, contentType)

	if err != nil {
		if err == InvalidContentType {
			http.Error(w, "No image was posted", http.StatusBadRequest)
		}
		serverError(w, err)
		return
	}

	photo := &Photo{Title: title,
		OwnerID:  user.ID,
		Filename: filename,
		Tags:     tags,
	}

	validator := getPhotoValidator(photo)

	if result, err := formHandler.Validate(validator); err != nil || !result.OK {
		if err != nil {
			serverError(w, err)
			return
		}
		result.Write(w)
		return
	}

	if err := photoMgr.Insert(photo); err != nil {
		serverError(w, err)
		return
	}

	sendMessage(&SocketMessage{user.Name, "", photo.ID, "photo_uploaded"})
	writeJSON(w, photo, http.StatusCreated)
}

func searchPhotos(w http.ResponseWriter, r *http.Request) {
	photos, err := photoMgr.Search(getPage(r), r.FormValue("q"))
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, photos, http.StatusOK)
}

func photosByOwnerID(c web.C, w http.ResponseWriter, r *http.Request) {
	ownerID, err := strconv.ParseInt(c.URLParams["ownerID"], 10, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	photos, err := photoMgr.ByOwnerID(getPage(r), ownerID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, photos, http.StatusOK)
}

func getPhotos(w http.ResponseWriter, r *http.Request) {
	photos, err := photoMgr.All(getPage(r), r.FormValue("orderBy"))
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, photos, http.StatusOK)
}

func getTags(w http.ResponseWriter, r *http.Request) {
	tags, err := photoMgr.GetTagCounts()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, tags, http.StatusOK)
}

func voteDown(c web.C, w http.ResponseWriter, r *http.Request) {
	vote(c, w, r, func(photo *Photo) { photo.DownVotes += 1 })
}

func voteUp(c web.C, w http.ResponseWriter, r *http.Request) {
	vote(c, w, r, func(photo *Photo) { photo.UpVotes += 1 })
}

func vote(c web.C, w http.ResponseWriter, r *http.Request, fn func(photo *Photo)) {
	var (
		photo *Photo
		err   error
	)
	user, ok := getUserOr401(w, r)
	if !ok {
		return
	}

	photo, ok = getPhotoOr404(c, w, r)
	if !ok {
		return
	}

	if !photo.CanVote(user) {
		http.Error(w, "You can't vote on this photo", http.StatusForbidden)
		return
	}

	fn(photo)

	if err = photoMgr.Update(photo); err != nil {
		serverError(w, err)
		return
	}

	user.RegisterVote(photo.ID)

	if err = userMgr.Update(user); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
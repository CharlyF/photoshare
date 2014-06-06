package routes

import (
	"github.com/danjac/photoshare/api/models"
	"github.com/danjac/photoshare/api/render"
	"github.com/danjac/photoshare/api/session"
	"github.com/danjac/photoshare/api/utils"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func deletePhoto(w http.ResponseWriter, r *http.Request) error {

	user, err := session.GetCurrentUser(r)
	if err != nil {
		return err
	}

	if user == nil {
		render.Status(w, http.StatusUnauthorized, "You can't do that")
		return err
	}

	photo, err := models.GetPhoto(mux.Vars(r)["id"])
	if err != nil {
		return err
	}
	if photo == nil {
		return render.Status(w, http.StatusNotFound, "Photo not found")
	}

	if !photo.CanDelete(user) {
		return render.Status(w, http.StatusForbidden, "You can't delete this photo")
	}
	if err := photo.Delete(); err != nil {
		return err
	}

	return render.Status(w, http.StatusOK, "Photo deleted")
}

func photoDetail(w http.ResponseWriter, r *http.Request) error {

	photo, err := models.GetPhotoDetail(mux.Vars(r)["id"])
	if err != nil {
		return err
	}
	if photo == nil {
		return render.Status(w, http.StatusNotFound, "Photo not found")
	}

	return render.JSON(w, http.StatusOK, photo)
}

func upload(w http.ResponseWriter, r *http.Request) error {

	user, err := session.GetCurrentUser(r)
	if err != nil {
		return err
	}
	if user == nil {
		return render.Status(w, http.StatusUnauthorized, "Not logged in")
	}

	title := r.FormValue("title")
	src, hdr, err := r.FormFile("photo")
	if err != nil {
		if err == http.ErrMissingFile {
			return render.Status(w, http.StatusBadRequest, "No image was posted")
		}
		return err
	}
	contentType := hdr.Header["Content-Type"][0]
	if contentType != "image/png" && contentType != "image/jpeg" {
		return render.Status(w, http.StatusBadRequest, "Not a valid image")
	}

	defer src.Close()
	filename, err := utils.ProcessImage(src, contentType)
	if err != nil {
		return err
	}

	photo := &models.Photo{Title: title,
		OwnerID: user.ID, Photo: filename}

	if result := photo.Validate(); !result.OK {
		return render.JSON(w, http.StatusBadRequest, result)
	}

	if err := photo.Save(); err != nil {
		return err
	}

	return render.JSON(w, http.StatusOK, photo)
}

func getPhotos(w http.ResponseWriter, r *http.Request) error {

	pageNum, err := strconv.ParseInt(r.FormValue("page"), 10, 0)
	if err != nil {
		pageNum = 1
	}

	photos, err := models.GetPhotos(pageNum)
	if err != nil {
		return err
	}
	return render.JSON(w, http.StatusOK, photos)
}

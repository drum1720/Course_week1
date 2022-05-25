package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

func (m *MyApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		m.profile(rw, r)
	case "/user/create":
		m.create(rw, r)
	default:
		responseError(rw, ApiError{HTTPStatus:http.StatusNotFound,Err: errors.New("unknown method")})
	}
}

func (o *OtherApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		o.create(rw, r)
	default:
		responseError(rw, ApiError{HTTPStatus:http.StatusNotFound,Err: errors.New("unknown method")})
	}
}

func (m *MyApi) profile(rw http.ResponseWriter, r *http.Request) {
	profileparams := ProfileParams{}
	if err := profileparams.FilingAndValidate(r); err != nil {
		responseError(rw, ApiError{HTTPStatus: http.StatusBadRequest,Err:err})
		return
	}
	response, err := m.Profile(nil, profileparams)
	if err != nil {
		apiErr, ok := err.(ApiError)
		if ok {
			responseError(rw, apiErr)
			return
		}
		responseError(rw, ApiError{HTTPStatus: http.StatusInternalServerError, Err:err})
		return
	}
	responseResult(rw, err, response)
}

func (m *MyApi) create(rw http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		responseError(rw, ApiError{HTTPStatus: http.StatusForbidden, Err: errors.New("unauthorized")})
		return
	}
	if r.Method != "POST" {
		responseError(rw, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: errors.New("bad method")})
		return
	}
	createparams := CreateParams{}
	if err := createparams.FilingAndValidate(r); err != nil {
		responseError(rw, ApiError{HTTPStatus: http.StatusBadRequest,Err:err})
		return
	}
	response, err := m.Create(nil, createparams)
	if err != nil {
		apiErr, ok := err.(ApiError)
		if ok {
			responseError(rw, apiErr)
			return
		}
		responseError(rw, ApiError{HTTPStatus: http.StatusInternalServerError, Err:err})
		return
	}
	responseResult(rw, err, response)
}

func (o *OtherApi) create(rw http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		responseError(rw, ApiError{HTTPStatus: http.StatusForbidden, Err: errors.New("unauthorized")})
		return
	}
	if r.Method != "POST" {
		responseError(rw, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: errors.New("bad method")})
		return
	}
	othercreateparams := OtherCreateParams{}
	if err := othercreateparams.FilingAndValidate(r); err != nil {
		responseError(rw, ApiError{HTTPStatus: http.StatusBadRequest,Err:err})
		return
	}
	response, err := o.Create(nil, othercreateparams)
	if err != nil {
		apiErr, ok := err.(ApiError)
		if ok {
			responseError(rw, apiErr)
			return
		}
		responseError(rw, ApiError{HTTPStatus: http.StatusInternalServerError, Err:err})
		return
	}
	responseResult(rw, err, response)
}

func responseError(rw http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	type CR map[string]interface{}

	apiErr, ok := err.(ApiError)
	if ok {
		rw.WriteHeader(apiErr.HTTPStatus)
	}
	responseMap := CR{"error": err.Error()}
	response, _ := json.Marshal(responseMap)
	rw.Write(response)
}

func responseResult(rw http.ResponseWriter, err error, result interface{}) {
	type CR map[string]interface{}
	textErr := ""

	if err != nil {
		textErr = err.Error()
	}

	responseMap := CR{
		"error":    textErr,
		"response": result,
	}

	response, err := json.Marshal(responseMap)
	if err != nil {
		fmt.Println(err)
	}
	rw.Write(response)
}

func (p *ProfileParams) FilingAndValidate(r *http.Request) error {
	var err error
	fmt.Println(err)
p.Login = r.FormValue("login")
if p.Login == ""{
return errors.New("login must me not empty")
}
	return nil
}

func (c *CreateParams) FilingAndValidate(r *http.Request) error {
	var err error
	fmt.Println(err)
c.Login = r.FormValue("login")
if c.Login == ""{
return errors.New("login must me not empty")
}
if len(c.Login) < 10{
return errors.New("login len must be >= 10")
}
c.Name = r.FormValue("name")
c.Name = r.FormValue("full_name")
c.Status = r.FormValue("status")
if c.Status == ""{
c.Status = "user"
}
isTrue:=false
if c.Status == "user"{
isTrue=true
}
if c.Status == "moderator"{
isTrue=true
}
if c.Status == "admin"{
isTrue=true
}
if !isTrue{
return errors.New("status must be one of [user, moderator, admin]")
}
c.Age,err = strconv.Atoi(r.FormValue("age"))
if err != nil{
return errors.New("age must be int")
}
if c.Age < 0{
return errors.New("age must be >= 0")
}
if c.Age > 128{
return errors.New("age must be <= 128")
}
	return nil
}

func (o *OtherCreateParams) FilingAndValidate(r *http.Request) error {
	var err error
	fmt.Println(err)
o.Username = r.FormValue("username")
if o.Username == ""{
return errors.New("username must me not empty")
}
if len(o.Username) < 3{
return errors.New("username len must be >= 3")
}
o.Name = r.FormValue("name")
o.Name = r.FormValue("account_name")
o.Class = r.FormValue("class")
isTrue:=false
if o.Class == "warrior"{
isTrue=true
}
if o.Class == "sorcerer"{
isTrue=true
}
if o.Class == "rouge"{
isTrue=true
}
if !isTrue{
return errors.New("class must be one of [warrior, sorcerer, rouge]")
}
if o.Class == ""{
o.Class = "warrior"
}
o.Level,err = strconv.Atoi(r.FormValue("level"))
if err != nil{
return errors.New("age must be int")
}
if o.Level < 1{
return errors.New("level must be >= 1")
}
if o.Level > 50{
return errors.New("level must be <= 50")
}
	return nil
}


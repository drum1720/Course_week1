package main

//func (ma *MyApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
//	switch r.URL.Path {
//	case "/user/profile":
//		ma.profile(rw, r)
//	case "/user/create":
//
//	default:
//		rw.WriteHeader(http.StatusNotFound)
//		responseError(rw, errors.New("unknown method"))
//	}
//}

//func (ma *MyApi) profile(rw http.ResponseWriter, r *http.Request) {
//	profileParams := ProfileParams{}
//
//	profileParams.Login = r.FormValue("login")
//
//	if err := profileParams.Validate(); err != nil {
//		responseError(rw, err)
//		return
//	}
//
//	user, err := ma.Profile(nil, profileParams)
//	if err != nil {
//		responseError(rw, err)
//		return
//	}
//
//	responseResult(rw, err, user)
//}

//
//func (p *ProfileParams) Validate() error {
//	if p.Login == "" {
//		return ApiError{
//			HTTPStatus: http.StatusBadRequest,
//			Err:        fmt.Errorf("login must me not empty"),
//		}
//	}
//	return nil
//}

//func responseResult(rw http.ResponseWriter, err error, result interface{}) {
//	type CR map[string]interface{}
//	textErr := ""
//
//	if err != nil {
//		textErr = err.Error()
//	}
//
//	responseMap := CR{
//		"error":    textErr,
//		"response": result,
//	}
//
//	response, err := json.Marshal(responseMap)
//	if err != nil {
//		fmt.Println(err)
//	}
//	rw.Write(response)
//}
//
//func responseError(rw http.ResponseWriter, err error) {
//	if err == nil {
//		return
//	}
//
//	type CR map[string]interface{}
//
//	apiErr, ok := err.(ApiError)
//	if ok {
//		rw.WriteHeader(apiErr.HTTPStatus)
//	}
//	rw.WriteHeader(http.StatusInternalServerError)
//	responseMap := CR{"error": err.Error()}
//	response, _ := json.Marshal(responseMap)
//	rw.Write(response)
//}

//func (oa *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	fmt.Println("Work_It")
//}

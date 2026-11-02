package entity

type ScrydeCaptchaResponse struct {
	Image               string `json:"image"`
	TemplateCoordinates []int  `json:"template_coordinates"`
	InputCoordinates    []int  `json:"input_coordinates"`
	SubmitCoordinates   []int  `json:"submit_coordinates"`
}

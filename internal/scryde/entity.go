package scryde

type Response struct {
	TemplateCoordinates []int `json:"template_coordinates"`
	InputCoordinates    []int `json:"input_coordinates"`
	SubmitCoordinates   []int `json:"submit_coordinates"`
}

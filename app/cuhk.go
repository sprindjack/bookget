package app

type Cuhk struct {
	dt *DownloadTask
}
type ResponsePage struct {
	ImagePage []ImagePage `json:"pages"`
}

type ImagePage struct {
	Pid        string `json:"pid"`
	Page       string `json:"page"`
	Label      string `json:"label"`
	Width      string `json:"width"`
	Height     string `json:"height"`
	Dsid       string `json:"dsid"`
	Token      string `json:"token"`
	Identifier string `json:"identifier"`
}

func (r *Cuhk) Init(iTask int, sUrl string) (msg string, err error) {
	//TODO implement me
	panic("implement me")
}

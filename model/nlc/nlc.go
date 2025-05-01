package nlc

// 基础响应结构
type BaseResponse struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
	Data Node   `json:"data"`
}

// 数据项结构
type DataItem struct {
	OrderSeq    string `json:"orderSeq"`
	ImageId     string `json:"imageId"`
	StructureId int    `json:"structureId"`
	PageNum     int    `json:"pageNum"`
}

// 通用节点结构（用于多级嵌套）
type Node struct {
	ImageIdList []DataItem `json:"imageIdList"`
	Total       int        `json:"total"`
}

type ImageData struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
	Data struct {
		OrderSeq        int     `json:"orderSeq"`
		ImageWidth      float64 `json:"imageWidth"`
		EntityTagStatus int     `json:"entityTagStatus"`
		FileName        string  `json:"fileName"`
		RecoErrorMsg    string  `json:"recoErrorMsg"`
		ImageId         int     `json:"imageId"`
		FilePath        string  `json:"filePath"`
		TxtFileName     string  `json:"txtFileName"`
		StructureId     int     `json:"structureId"`
		UpdateTime      string  `json:"updateTime"`
		DelFlag         int     `json:"delFlag"`
		RecoJson        struct {
			Chars []interface{} `json:"chars"`
		} `json:"recoJson"`
		SeriesId    int     `json:"seriesId"`
		ImageHeight float64 `json:"imageHeight"`
		FtpId       int     `json:"ftpId"`
		MetadataId  int     `json:"metadataId"`
		Comma       string  `json:"comma"`
		CreateTime  string  `json:"createTime"`
		PdfFileName string  `json:"pdfFileName"`
		FileType    string  `json:"fileType"`
		RecoStatus  int     `json:"recoStatus"`
		Trans       string  `json:"trans"`
	} `json:"data"`
}

package models

type TierList struct {
	Title         string      `json:"title"`
	BgColor       string      `json:"bg_color"`
	Rows          []TierRow   `json:"rows"`
	StagingImages []ImageItem `json:"staging_images"`
}

type TierRow struct {
	ID     string      `json:"id"`
	Label  string      `json:"label"`
	Color  string      `json:"color"`
	Images []ImageItem `json:"images"`
}

type ImageItem struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type Message struct {
	Type string `json:"type"`

	// set_title / title_updated
	Title string `json:"title,omitempty"`

	// set_bg_color / bg_color_updated
	Color string `json:"color,omitempty"`

	// update_label / label_updated
	RowID string `json:"row_id,omitempty"`
	Label string `json:"label,omitempty"`

	// add_row
	Position        string `json:"position,omitempty"`
	RelativeToRowID string `json:"relative_to_row_id,omitempty"`

	// row_added
	Row  *TierRow  `json:"row,omitempty"`
	Rows []TierRow `json:"rows,omitempty"`

	// row_deleted / row_cleared
	StagingImages []ImageItem `json:"staging_images,omitempty"`

	// move_row
	Direction string `json:"direction,omitempty"`

	// upload_image
	Filename string `json:"filename,omitempty"`
	Data     string `json:"data,omitempty"`

	// image_uploaded
	Image *ImageItem `json:"image,omitempty"`

	// move_image
	ImageID     string `json:"image_id,omitempty"`
	TargetRowID string `json:"target_row_id,omitempty"`
	TargetIndex int    `json:"target_index,omitempty"`
}

type FullStateMsg struct {
	Type          string      `json:"type"`
	Title         string      `json:"title"`
	BgColor       string      `json:"bg_color"`
	Rows          []TierRow   `json:"rows"`
	StagingImages []ImageItem `json:"staging_images"`
}


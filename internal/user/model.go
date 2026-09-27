package user

type User struct {
	ID      string   `bson:"_id" json:"id"`
	Name    string   `bson:"name" json:"name"`
	Country string   `bson:"country" json:"country"`
	Plan    string   `bson:"plan" json:"plan"`
	Tags    []string `bson:"tags" json:"tags"`
}

type CreateUserRequest struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Country string   `json:"country"`
	Plan    string   `json:"plan"`
	Tags    []string `json:"tags"`
}

type UpdateUserRequest struct {
	Name    *string   `json:"name,omitempty"`
	Country *string   `json:"country,omitempty"`
	Plan    *string   `json:"plan,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
}

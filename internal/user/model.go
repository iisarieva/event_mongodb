package user

type User struct {
	ID      string `bson:"_id" json:"id"`
	Name    string `bson:"name" json:"name"`
	Country string `bson:"country" json:"country"`
	Plan    string `bson:"plan" json:"plan"`
}

type CreateUserRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Plan    string `json:"plan"`
}

type UpdateUserRequest struct {
	Name    *string `json:"name,omitempty"`
	Country *string `json:"country,omitempty"`
	Plan    *string `json:"plan,omitempty"`
}

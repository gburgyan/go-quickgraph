package examples

import (
	"context"
	"strings"
)

type Course struct {
	Title      string  `json:"title"`
	Instructor string  `json:"instructor"`
	Price      float64 `json:"price"`
}

var courses = []*Course{
	{
		Title:      "Golang",
		Instructor: "John Doe",
		Price:      10.99,
	},
	{
		Title:      "Python",
		Instructor: "Jane Doe",
		Price:      12.99,
	},
	{
		Title:      "Java",
		Instructor: "Jack Doe",
		Price:      11.99,
	},
	{
		Title:      "Ruby",
		Instructor: "Jill Doe",
		Price:      9.99,
	},
	{
		Title:      "C++",
		Instructor: "James Doe",
		Price:      8.99,
	},
	{
		Title:      "C#",
		Instructor: "Judy Doe",
		Price:      7.99,
	},
}

func GetCourses(ctx context.Context, categories []*string) []*Course {
	if len(categories) == 0 {
		return courses
	}

	filteredCourses := []*Course{}
	for _, course := range courses {
		for _, category := range categories {
			if strings.Contains(course.Title, *category) {
				filteredCourses = append(filteredCourses, course)
			}
		}
	}
	return filteredCourses
}

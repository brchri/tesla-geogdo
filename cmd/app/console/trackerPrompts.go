package console

import (
	"fmt"
	"strconv"

	asciiArt "github.com/common-nighthawk/go-figure"
)

func runTrackerPrompts() []interface{} {
	type Tracker struct {
		Id           interface{} `yaml:"id"`
		LatTopic     string      `yaml:"lat_topic,omitempty"`
		LngTopic     string      `yaml:"lng_topic,omitempty"`
		ComplexTopic struct {
			Topic      string `yaml:"topic,omitempty"`
			LatJsonKey string `yaml:"lat_json_key,omitempty"`
			LngJsonKey string `yaml:"lng_json_key,omitempty"`
		} `yaml:"complex_topic,omitempty"`
	}

	asciiArt.NewFigure("Trackers", "", false).Print()
	fmt.Print("\nWe will now configure trackers for this garage door\n\n")

	trackers := []interface{}{}

	for {
		tracker := Tracker{}
		response := promptUser(question{
			prompt:                 "Please enter an ID for this tracker. It must be unique from all other trackers, and can be any letter or number combination",
			validResponseRegex:     "^\\w+$",
			invalidResponseMessage: "Please enter a valid combination of letters and/or numbers",
		})
		trackerInt, err := strconv.Atoi(response)
		if err == nil {
			tracker.Id = trackerInt
		} else {
			tracker.Id = response
		}
		response = promptUser(question{
			prompt:                 "Does your tracker use simple topics (e.g. basic lat and long values are published to separate topics) or a complex topic (e.g. lat and long are published to the same topic in a json structure)? [s|c]\ns: simple\nc: complex",
			validResponseRegex:     "^(c|s)$",
			invalidResponseMessage: "Please enter s (for simple topic) or c (for complex topic)",
		})
		if response == "c" {
			tracker.ComplexTopic.Topic = promptUser(question{
				prompt:             "Please enter the complex topic where lat and long information is published (e.g. my/complex/topic)",
				validResponseRegex: ".+",
			})
			tracker.ComplexTopic.LatJsonKey = promptUser(question{
				prompt:             "Please enter the top-level json key for latitude (e.g. if the json structure is {lat: 123.123, lng: -123.123}, then you should enter 'lat' here). Note: nested json keys are not supported, only top-level keys: ",
				validResponseRegex: ".+",
			})
			tracker.ComplexTopic.LngJsonKey = promptUser(question{
				prompt:             "Please enter the top-level json key for longitude (e.g. if the json structure is {lat: 123.123, lng: -123.123}, then you should enter 'lng' here). Note: nested json keys are not supported, only top-level keys: ",
				validResponseRegex: ".+",
			})
		} else {
			tracker.LatTopic = promptUser(question{
				prompt:             "Please enter the topic where latitude is published: ",
				validResponseRegex: ".+",
			})
			tracker.LngTopic = promptUser(question{
				prompt:             "Please enter the topic where longitude is published: ",
				validResponseRegex: ".+",
			})
		}

		trackers = append(trackers, tracker)
		response = promptUser(question{
			prompt:                 "Would you like to add more trackers to this garage door? [y|n]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
		})
		if isNoRegex.MatchString(response) {
			break
		}
	}

	fmt.Println("Done creating trackers for this garage door, moving on...")

	return trackers
}

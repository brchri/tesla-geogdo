package console

import (
	"fmt"
	"regexp"
)

func runTrackerPrompts() []interface{} {
	trackers := []interface{}{}
	re := regexp.MustCompile("n|N")

	for {
		tracker := map[string]interface{}{}
		response := promptUser(question{
			prompt:                 "Please enter an ID for this tracker. It must be unique from all other trackers, and can be any letter or number combination",
			validResponseRegex:     "^\\w+$",
			invalidResponseMessage: "Please enter a valid combination of letters and/or numbers",
		})
		tracker["id"] = response
		response = promptUser(question{
			prompt:                 "Do your tracker use simple topics (e.g. basic lat and long values are published to separate topics) or a complex topic (e.g. lat and long are published to the same topic in a json structure)? [s|c]\ns: simple\nc: complex",
			validResponseRegex:     "^(c|s)$",
			invalidResponseMessage: "Please enter s (for simple topic) or c (for complex topic)",
		})
		if response == "c" {
			complexTopic := map[string]string{}
			response = promptUser(question{
				prompt:             "Please enter the complex topic where lat and long information is published (e.g. my/complex/topic)",
				validResponseRegex: ".+",
			})
			complexTopic["topic"] = response
			response = promptUser(question{
				prompt:             "Please enter the top-level json key for latitude (e.g. if the json structure is {lat: 123.123, lng: -123.123}, then you should enter 'lat' here). Note: nested json keys are not supported, only top-level keys: ",
				validResponseRegex: ".+",
			})
			complexTopic["lat_json_key"] = response
			response = promptUser(question{
				prompt:             "Please enter the top-level json key for longitude (e.g. if the json structure is {lat: 123.123, lng: -123.123}, then you should enter 'lng' here). Note: nested json keys are not supported, only top-level keys: ",
				validResponseRegex: ".+",
			})
			complexTopic["lng_json_key"] = response
			tracker["complex_topic"] = complexTopic
		} else {
			response = promptUser(question{
				prompt:             "Please enter the topic where latitude is published: ",
				validResponseRegex: ".+",
			})
			tracker["lat_topic"] = response
			response = promptUser(question{
				prompt:             "Please enter the topic where longitude is published: ",
				validResponseRegex: ".+",
			})
			tracker["lng_topic"] = response
		}

		trackers = append(trackers, tracker)
		response = promptUser(question{
			prompt:                 "Would you like to add more trackers to this garage door? [y|n]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
		})
		if re.MatchString(response) {
			break
		}
	}

	fmt.Println("Done creating trackers for this garage door, moving on...")

	return trackers
}

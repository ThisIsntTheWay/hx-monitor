package transcript

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thisisnttheway/hx-checker/models"
)

var _thisYear int = time.Now().Year()

// Parses the provided transcript to an AirspaceStatus
func parseAirspaceStates(transcript string) models.AirspaceStatus {
	// Default areas (Meiringen)
	areas := []models.Area{
		{0, false}, // CTR
		{1, false}, // TMA x
		{2, false},
		{3, false},
		{4, false},
		{5, false},
		{6, false},
	}

	transcript = strings.ToLower(transcript)

	// Correct common twilio transcription mistakes
	transcript = strings.Replace(transcript, "my ring", "meiringen", -1)
	transcript = strings.Replace(transcript, "pma", "tma", -1)
	transcript = strings.Replace(transcript, "be act again", "be active again", -1)
	transcript = strings.Replace(transcript, "the activated", "deactivated", -1)

	// If true, then transcript is from a time outside flight operating hours
	// As such, all mentioned sectors are inactive
	canBeActivated := strings.Contains(transcript, "can be")

	// If true, then likely no areas will be active (weekend transcript)
	hasMultipleCtrSubstrings := strings.Count(transcript, "ctr") > 1
	var ctrSubstringIndex int
	if hasMultipleCtrSubstrings {
		ctrSubstringIndex = 0
	} else {
		ctrSubstringIndex = 1
	}

	// First split by CTR, then by keyword "active"
	splitCtr := strings.Split(transcript, "ctr")
	splitActive := strings.Split(splitCtr[ctrSubstringIndex], "active")

	// If keyword occurs in the first segment, then no areas will be active
	hasAreNotActive := strings.Contains(transcript, "are not active")

	everyTmaTargeted := strings.Contains(splitActive[0], "all tma")

	if !canBeActivated && !everyTmaTargeted && !hasAreNotActive {
		// CTR and specific TMAs are active
		activeTmas := regexp.MustCompile(`\d`).FindAllString(splitActive[0], -1)

		for i := range activeTmas {
			areas[i+1].Status = true
		}

		// CTR
		areas[0].Status = true
	} else if !hasAreNotActive && everyTmaTargeted {
		// Eveything is active
		for i := range areas {
			areas[i].Status = true
		}
	} else if hasAreNotActive {
		// Everything is inactive, therefore preserve defaults
	}

	return models.AirspaceStatus{
		Areas:      areas,
		NextUpdate: time.Unix(0, 0),
	}
}

// Get the current date but set hours and minutes of an arbitrary timeString
func parseTimeToCurrentDate(timeString string) (time.Time, error) {
	parsedTime, err := time.Parse("15:04", timeString)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing time: %w", err)
	}

	now := time.Now()

	finalTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		parsedTime.Hour(), parsedTime.Minute(), 0, 0,
		now.Location(),
	)

	return finalTime, nil
}

// Extract time segments; Next updates and flight operating hours
func parseTimeSegments(transcript string) []models.TimeSegment {
	// \d{3,4} can also falsely match years - will be handled below
	patternTimeSegments := `\d{1,2}[: ]\d{2}|\d{3,4}`

	// Split all time segments by the "local time" substring.
	// Segment 1: Message update times,
	// Segment 2: Flight operating hours morning,
	// Segment 3: Flight operating hours evening,
	// Format: [[t1, t2], [t1, t2], [t1, t2]]
	var timeSegments [][]time.Time

	// Check if this transcript is for the weekend (only one update time)
	// Sometimes gets misinterpreted as "be act again"
	onlyOneUpdateTime := strings.Contains(transcript, "be active again")

	splitLocalTime := strings.Split(transcript, "local time")
	for i, split := range splitLocalTime {
		trimmed := strings.TrimSpace(split)
		if onlyOneUpdateTime && i > 0 {
			break
		}

		if regexp.MustCompile(`\d`).MatchString(trimmed) {
			re := regexp.MustCompile(patternTimeSegments)
			matches := re.FindAllStringSubmatch(trimmed, -1)

			// Parse each time segment within local time segment, but only if we have any matches at alls
			if len(matches) < 1 {
				continue
			}

			var segments []time.Time
			for _, match := range matches {
				isYear := false

				// Prevent years from being interpreted as times
				// Years will likely be prepended by the string "[of] month"
				matchYear := regexp.MustCompile(`2\d{1}\d{2}`).FindAllString(match[0], -1)
				if len(matchYear) > 0 {
					reg := fmt.Sprintf("(of(\\s)?)?\\w+ %s", matchYear[0])
					re := regexp.MustCompile(reg)
					matchYearContext := re.FindAllString(trimmed, -1)

					if len(matchYearContext) > 0 {
						// Prevent false positive (e.g. "1305 being interpreted as a year")
						if y, err := strconv.Atoi(matchYear[0]); err == nil && y >= _thisYear {
							// If string converts succesfully, then this is absolutely a year
							// Meiringen will always say "... <month> 2024 ..."
							_, err := time.Parse("January 2006", matchYearContext[0])
							isYear = err == nil
						}

						slog.Debug(
							"PARSER",
							"matchYear", matchYear,
							"matchYearContext", matchYearContext,
							"isYear", isYear,
						)
					}
				}

				if isYear {
					continue
				}

				// Transform 730 -> 7 30 | 1305 -> 13 05
				// In case of len(s) == 3, this will naively assume that the first digit is the hour
				var transformedString string = match[0]
				if !strings.Contains(match[0], ":") && !strings.Contains(match[0], " ") {
					if len(transformedString) == 3 {
						transformedString = fmt.Sprintf(
							"%s %s",
							string(transformedString[0]),
							string(transformedString[1:]),
						)
					} else if len(transformedString) == 4 {
						transformedString = fmt.Sprintf(
							"%s %s",
							string(transformedString[0:2]),
							string(transformedString[2:]),
						)
					}
				}

				replacedString := strings.Replace(transformedString, " ", ":", 1)
				convertedTime, err := parseTimeToCurrentDate(replacedString)
				if err != nil {
					panic(err)
				}

				segments = append(segments, convertedTime)
			}

			timeSegments = append(timeSegments, segments)
		}
	}

	var rO []models.TimeSegment

	// If yes, then the update time will most likely be on a future day
	if onlyOneUpdateTime {
		// Check if the transcript contains a concrete date
		// Happens occasionally when the military is on "vacation"
		re := regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th) of (\w+) (\d{4})`)
		m := re.FindStringSubmatch(transcript)
		if len(m) == 4 {
			var parsedDate time.Time
			var processedDate time.Time
			slog.Info("PARSER", "action", "parseTimeSegments", "message", "Transcript seems to contain concrete date")
			parsedDate, err := time.Parse(
				"2 January 2006",
				fmt.Sprintf("%s %s %s", m[1], m[2], m[3]),
			)
			if err != nil {
				processedDate = time.Time{}
			} else {
				h, m := timeSegments[0][0].Hour(), timeSegments[0][0].Minute()
				processedDate = parsedDate
				processedDate = processedDate.Add(time.Hour * time.Duration(h))
				processedDate = processedDate.Add(time.Minute * time.Duration(m))
			}

			timeSegments[0][0] = processedDate
		} else {
			// ToDo: handle next update time not necessarily being on monday
			// Haven't seen this in Meiringens transcript yet
			daysUntilMonday := (int(time.Monday) - int(timeSegments[0][0].Weekday()) + 7) % 7
			if daysUntilMonday == 0 {
				daysUntilMonday = 7
			}

			timeSegments[0][0] = timeSegments[0][0].AddDate(0, 0, daysUntilMonday)
		}
	} else {
		var operatingHours []time.Time
		for i := range timeSegments[1 : len(timeSegments)-1] {
			operatingHours = append(operatingHours, timeSegments[i]...)
		}

		rO = append(rO, models.TimeSegment{Type: "OperatingHours", Times: operatingHours})
	}

	if len(timeSegments) == 0 {
		slog.Error("PARSER", "action", "parseTimeSegments", "gotTimeSegments", false, "transcript", transcript)
		return nil
	}

	rO = append(rO, models.TimeSegment{Type: "UpdateTimes", Times: timeSegments[0]})
	return rO
}

// Parse a transcript based on a reference time
func ParseTranscript(transcript string, referenceTime time.Time) models.AirspaceStatus {
	slog.Info("PARSER", "event", "startParse", "transcript", transcript, "referenceTime", referenceTime)

	var timeSegments []models.TimeSegment
	var airspaceState models.AirspaceStatus

	timeSegments = parseTimeSegments(transcript)
	airspaceState = parseAirspaceStates(transcript)

	// Assign time segments
	var updateTimeTimeSegment models.TimeSegment
	var operatingHoursTimeSegment models.TimeSegment
	slog.Debug("PARSER", "action", "assembleTimeSegments", "timeSegments", timeSegments)
	for _, segment := range timeSegments {
		if segment.Type == "UpdateTimes" {
			updateTimeTimeSegment = segment
		} else if segment.Type == "OperatingHours" {
			operatingHoursTimeSegment = segment
		}
	}

	nextUpdateTime := time.Time{}
	for _, segment := range updateTimeTimeSegment.Times {
		slog.Debug("PARSER", "action", "setUpdateTime", "candidateSegment", segment)
		if referenceTime.Before(segment) {
			slog.Debug("PARSER", "action", "setUpdateTimeFinal", "candidateSegment", segment)
			nextUpdateTime = segment
			break
		}
	}

	airspaceState.NextUpdate = nextUpdateTime
	airspaceState.OperatingHours = operatingHoursTimeSegment.Times

	slog.Info("PARSER", "event", "finishParse", "airspaceState", airspaceState)

	return airspaceState
}

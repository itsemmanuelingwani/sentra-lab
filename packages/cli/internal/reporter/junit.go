package reporter

import (
	"encoding/xml"
	"io"
	"time"
)

type JUnitReporter struct{}

func NewJUnitReporter() Reporter {
	return &JUnitReporter{}
}

func (jr *JUnitReporter) Report(w io.Writer, summary interface{}, results interface{}) error {
	testSuite := JUnitTestSuite{
		Name:      "Sentra Lab Tests",
		Tests:     0,
		Failures:  0,
		Errors:    0,
		Time:      0,
		Timestamp: time.Now().Format(time.RFC3339),
		TestCases: []JUnitTestCase{},
	}

	data, err := xml.MarshalIndent(testSuite, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

type JUnitTestSuite struct {
	XMLName   xml.Name         `xml:"testsuite"`
	Name      string           `xml:"name,attr"`
	Tests     int              `xml:"tests,attr"`
	Failures  int              `xml:"failures,attr"`
	Errors    int              `xml:"errors,attr"`
	Time      float64          `xml:"time,attr"`
	Timestamp string           `xml:"timestamp,attr"`
	TestCases []JUnitTestCase  `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string         `xml:"name,attr"`
	ClassName string         `xml:"classname,attr"`
	Time      float64        `xml:"time,attr"`
	Failure   *JUnitFailure  `xml:"failure,omitempty"`
}

type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}
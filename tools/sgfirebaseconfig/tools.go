package sgfirebaseconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.einride.tech/sage/sg"
)

type CSP struct {
	Rules map[string]Group
	Ctx   context.Context
}

type Group struct {
	ConnectSrc []string
	FontSrc    []string
	FrameSrc   []string
	ImgSrc     []string
	ScriptSrc  []string
	StyleSrc   []string
	WorkerSrc  []string
}

func Init(ctx context.Context) CSP {
	var csp CSP
	csp.Ctx = ctx
	csp.Rules = make(map[string]Group, 0)
	csp.AddRule("firebase-auth", Group{
		ConnectSrc: []string{
			"https://identitytoolkit.googleapis.com",
			"https://securetoken.googleapis.com",
		},
	})
	csp.AddRule("sentry", Group{
		ConnectSrc: []string{"https://*.ingest.sentry.io"},
		ScriptSrc:  []string{"https://sentry.io"},
	})
	csp.AddRule("amplitude", Group{
		ConnectSrc: []string{"https://api2.amplitude.com/2/httpapi"},
	})
	csp.AddRule("mapbox", Group{
		ConnectSrc: []string{"https://api.mapbox.com", "https://events.mapbox.com"},
		ScriptSrc:  []string{"https://api.mapbox.com"},
		ImgSrc:     []string{"data:"},
	})
	return csp
}

func (csp *CSP) AddRule(name string, rule Group) {
	csp.Rules[name] = rule
}

func (csp *CSP) toString(groups []string, reportToDomain string) string {
	var g Group
	for _, toAdd := range groups {
		if v, found := csp.Rules[toAdd]; found {
			g = Group{
				ConnectSrc: append(g.ConnectSrc, v.ConnectSrc...),
				FontSrc:    append(g.FontSrc, v.FontSrc...),
				FrameSrc:   append(g.FrameSrc, v.FrameSrc...),
				ImgSrc:     append(g.ImgSrc, v.ImgSrc...),
				ScriptSrc:  append(g.ScriptSrc, v.ScriptSrc...),
				StyleSrc:   append(g.StyleSrc, v.StyleSrc...),
				WorkerSrc:  append(g.WorkerSrc, v.WorkerSrc...),
			}
		}
	}
	return fmt.Sprintf(
		"default-src 'self';"+
			" connect-src %s;"+
			" font-src %s;"+
			" frame-src %s;"+
			" img-src %s;"+
			" script-src %s;"+
			" style-src %s;"+
			" worker-src %s;"+
			" report-uri %s;"+
			" report-to default",
		strings.Join(g.ConnectSrc, " "),
		strings.Join(g.FontSrc, " "),
		strings.Join(g.FrameSrc, " "),
		strings.Join(g.ImgSrc, " "),
		strings.Join(g.ScriptSrc, " "),
		strings.Join(g.StyleSrc, " "),
		strings.Join(g.WorkerSrc, " "),
		reportToDomain,
	)
}

func (csp *CSP) Update(filePath, siteName string, groups []string, reportToDomain string) error {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		return err
	}
	hosting, ok := result["hosting"].([]interface{})
	if !ok {
		return fmt.Errorf("couldn't type assert the hosting array")
	}
	// Iterate over each site in the array
	for hostIndex, host := range hosting {
		// type assert each site into the correct format
		site, ok := host.(map[string]interface{})
		if !ok {
			return fmt.Errorf("couldn't type assert the sites")
		}
		if site["site"] == siteName {
			sources, ok := site["headers"].([]interface{})
			if !ok {
				return fmt.Errorf("couldn't type assert %s", siteName)
			}
			for sourceIndex, source := range sources {
				headers := source.(map[string]interface{})["headers"]
				for headerIndex, header := range headers.([]interface{}) {
					if header.(map[string]interface{})["key"] == "Content-Security-Policy" {
						tmp := map[string]string{
							"key":   "Content-Security-Policy",
							"value": csp.toString(groups, reportToDomain),
						}
						sg.Logger(csp.Ctx).Printf("replacing csp for %s\n", siteName)
						//nolint:lll
						result["hosting"].([]interface{})[hostIndex].(map[string]interface{})["headers"].([]interface{})[sourceIndex].(map[string]interface{})["headers"].([]interface{})[headerIndex] = tmp
					}
				}
			}
		}
	}
	file, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, file, 0o600)
}

func (csp *CSP) Create(
	filePath string,
	siteName string,
	groups []string,
	reportToDomain string,
) error {
	firebaseConfig := fmt.Sprintf(firebaseConfigTemplate, siteName, csp.toString(groups, reportToDomain), reportToDomain)
	sg.Logger(csp.Ctx).Printf("creating csp for %s\n", siteName)
	return os.WriteFile(filePath, []byte(firebaseConfig), 0o600)
}

// A reasonable default firebase config.
const firebaseConfigTemplate = `{
   "hosting":[{
      "site": "%s",
      "public":"build",
      "ignore":[
         "firebase.json",
         "**/.*",
         "**/node_modules/**"
      ],
      "rewrites":[
         {
            "source":"**",
            "destination":"/index.html"
         }
      ],
      "headers":[
         {
            "source":"/**",
            "headers":[
               {
                  "key":"Cache-Control",
                  "value":"max-age=120"
               },
			   {
                  "key":"X-Frame-Options",
				  "value":"SAMEORIGIN"
			   },
			   {
                  "key":"Content-Security-Policy",
				  "value":"%s"
			   },
			   {
				  "key":"Referrer-Policy",
				  "value":"origin-when-cross-origin"
			   },
			   {
                   "key": "Report-To",
				   "value": "'group':'default', 'max_age':3600, 'endpoints':[{'url':'%s'}],'include_subdomains':true'"
			   }
            ]
         },
         {
            "source":"**/*.@(jpg|jpeg|gif|png|svg|webp|js|css|eot|otf|ttf|ttc|woff|woff2|font.css)",
            "headers":[
               {
                  "key":"Cache-Control",
                  "value":"max-age=604800"
               }
            ]
         }
      ]
   }]
}
`

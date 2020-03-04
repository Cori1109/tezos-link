// GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag at
// 2020-03-04 17:16:37.447472 +0100 CET m=+0.091346888

package docs

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/alecthomas/template"
	"github.com/swaggo/swag"
)

var doc = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{.Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "API Support",
            "email": "email@ded.fr"
        },
        "license": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/health": {
            "get": {
                "summary": "get application health",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.Health"
                        }
                    }
                }
            }
        },
        "/projects": {
            "post": {
                "produces": [
                    "application/json"
                ],
                "summary": "Create a Project",
                "parameters": [
                    {
                        "description": "New Project",
                        "name": "new-project",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/inputs.NewProject"
                        }
                    }
                ],
                "responses": {
                    "201": {},
                    "400": {}
                }
            }
        },
        "/projects/{uuid}": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "Get a Project with the associated metrics",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Project UUID",
                        "name": "uuid",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/outputs.ProjectOutputWithMetrics"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "inputs.NewProject": {
            "type": "object",
            "properties": {
                "title": {
                    "type": "string"
                }
            }
        },
        "model.Health": {
            "type": "object",
            "properties": {
                "connectedToDb": {
                    "type": "boolean"
                }
            }
        },
        "outputs.MetricsOutput": {
            "type": "object",
            "properties": {
                "requestsCount": {
                    "type": "integer"
                }
            }
        },
        "outputs.ProjectOutputWithMetrics": {
            "type": "object",
            "properties": {
                "metrics": {
                    "type": "object",
                    "$ref": "#/definitions/outputs.MetricsOutput"
                },
                "title": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        }
    }
}`

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = swaggerInfo{
	Version:     "v1",
	Host:        "",
	BasePath:    "/api/v1",
	Schemes:     []string{},
	Title:       "Tezos Link API",
	Description: "API to manage projects",
}

type s struct{}

func (s *s) ReadDoc() string {
	sInfo := SwaggerInfo
	sInfo.Description = strings.Replace(sInfo.Description, "\n", "\\n", -1)

	t, err := template.New("swagger_info").Funcs(template.FuncMap{
		"marshal": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
	}).Parse(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, sInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register(swag.Name, &s{})
}

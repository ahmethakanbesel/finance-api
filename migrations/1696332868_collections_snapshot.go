package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `[
			{
				"id": "_pb_users_auth_",
				"created": "2023-10-03 11:07:27.880Z",
				"updated": "2023-10-03 11:08:11.761Z",
				"name": "users",
				"type": "auth",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "users_name",
						"name": "name",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "users_avatar",
						"name": "avatar",
						"type": "file",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"maxSize": 5242880,
							"mimeTypes": [
								"image/jpeg",
								"image/png",
								"image/svg+xml",
								"image/gif",
								"image/webp"
							],
							"thumbs": null,
							"protected": false
						}
					}
				],
				"indexes": [],
				"listRule": "id = @request.auth.id",
				"viewRule": "id = @request.auth.id",
				"createRule": "",
				"updateRule": "id = @request.auth.id",
				"deleteRule": "id = @request.auth.id",
				"options": {
					"allowEmailAuth": true,
					"allowOAuth2Auth": true,
					"allowUsernameAuth": true,
					"exceptEmailDomains": null,
					"manageRule": null,
					"minPasswordLength": 8,
					"onlyEmailDomains": null,
					"requireEmail": false
				}
			},
			{
				"id": "jy0x3jni8yi3kl4",
				"created": "2023-10-03 11:08:11.761Z",
				"updated": "2023-10-03 11:08:11.761Z",
				"name": "prices",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "ajnolnwo",
						"name": "source",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "2rnso8oj",
						"name": "symbol",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "c40ovpvj",
						"name": "date",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "fie1pel2",
						"name": "closePrice",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "suyjmbhl",
						"name": "currency",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 3,
							"max": 3,
							"pattern": ""
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_JCfkyKe` + "`" + ` ON ` + "`" + `prices` + "`" + ` (\n  ` + "`" + `source` + "`" + `,\n  ` + "`" + `symbol` + "`" + `,\n  ` + "`" + `date` + "`" + `,\n  ` + "`" + `currency` + "`" + `\n)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "gpi42nw2nh0v9vf",
				"created": "2023-10-03 11:08:11.762Z",
				"updated": "2023-10-03 11:08:11.762Z",
				"name": "scrapes",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "wycrxfyk",
						"name": "source",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "wc7tkvuy",
						"name": "symbol",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 3,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "eoc3jfon",
						"name": "startDate",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "cggjl1no",
						"name": "endDate",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "qeqrewh1",
						"name": "currency",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 3,
							"max": 3,
							"pattern": ""
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_GjnFt9D` + "`" + ` ON ` + "`" + `scrapes` + "`" + ` (\n  ` + "`" + `source` + "`" + `,\n  ` + "`" + `symbol` + "`" + `,\n  ` + "`" + `startDate` + "`" + `,\n  ` + "`" + `endDate` + "`" + `,\n  ` + "`" + `currency` + "`" + `\n)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			}
		]`

		collections := []*models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			return err
		}

		return daos.New(db).ImportCollections(collections, true, nil)
	}, func(db dbx.Builder) error {
		return nil
	})
}

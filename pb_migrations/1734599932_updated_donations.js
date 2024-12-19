/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2848092154")

  // update collection data
  unmarshal({
    "updateRule": "@request.auth.id != \"\" "
  }, collection)

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select2063623452",
    "maxSelect": 1,
    "name": "status",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "pending_review",
      "in_review",
      "rejected",
      "accepted"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2848092154")

  // update collection data
  unmarshal({
    "updateRule": null
  }, collection)

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select2063623452",
    "maxSelect": 1,
    "name": "status",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "pending_review",
      "in_review",
      "accepted"
    ]
  }))

  return app.save(collection)
})

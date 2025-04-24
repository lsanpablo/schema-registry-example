# Events Directory
A catalog of all events. You can see a full index at [index.md](./index.md).

# Service Directory
A grpc service that takes in a payload in bytes and an event schema ID.
It validates the payload and returns a valid/not valid return value.

# Tools
All tools are built using nix. So from the root directory you can run `nix develop` and all of them will be available to you.

`event-template` - Takes in an eventID that looks like a NATS subject and a version. It creates a documentation template in the correct directory.

Example Usage: `event-tempate user.{user_id}.created v1`

`generate-index` - Walks the `./events` directory and generates the `index.md` catalog with all events and links to the documentation

`generate-go-types` - It takes in a JSON Schema file and generates a Go struct type with the correct types. It doesn't add
validation.

Example Usage: `generate-go-types -s events/user/\{user_id\}/created/v1.schema.json -o test.go -n customers`
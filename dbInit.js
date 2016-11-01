// Initialise the database on first run

var conn = new Mongo();
var db = conn.getDB("admin");

// Insert the default admin user
db.createUser({
  user: "admin",
  pwd: "admin",
  roles: [{ role: "userAdminAnyDatabase", db: "admin" }]
});

db = conn.getDB("scms");

// Insert the scms user into the newly created scms database
db.createUser({
  user: "scms",
  pwd: "scms",
  roles: [{ role: "readWrite", db: "scms" }]
});

// Insert the default admin user so that people can login
db.users.insert({
  name: "admin",
  email: "admin@email.com",
  password: "admin",
  level: 7,
  admin: true
});

// insert an example document so people can see what it looks like
db.pages.insert({
  title: "Example Document",
  body: "This is an example document. Please edit it",
  url: "",
  level: 0
});
// Initialise the database on first run

var conn = new Mongo();
var db = conn.getDB("admin");

// Insert the default admin user
db.createUser({
  user: "admin",
  pwd: "admin",
  roles: [{ role: "userAdminAnyDatabase", db: "admin" }]
});

// create the log database and its things
db = conn.getDB("scms_log");

db.createUser({
  user: "scms_log",
  pwd: "scms_log",
  roles: [{ role: "read", db: "scms_log" }]
});

// Create the default "scms" database and its things
db = conn.getDB("scms");

// Insert the scms user into the newly created scms database
db.createUser({
  user: "scms",
  pwd: "scms",
  roles: [
    { role: "readWrite", db: "scms" },
    { role: "readWrite", db: "scms_log"}
  ]
});

// Insert the default admin user so that people can login
db.users.insert({
  name: "admin",
  email: "admin@email.com",
  password: "password",
  level: 7,
  admin: true,
  tech: false
});

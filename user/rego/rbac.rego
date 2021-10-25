package rbac.authz

default allow = false

allow {
	is_admin
}

admins = ["bojan"]
members = ["bojan2"]

is_admin {
    admins[_] == input.username
}

is_member {
    members[_] == input.username
}
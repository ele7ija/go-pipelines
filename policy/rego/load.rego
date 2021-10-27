package load.image_requests

default allow = false

MAX_NUMBER  = 100
MAX_SIZE    = 100000000 # 100 MB

allow {
    number_less
    size_less
}

number_less {
    n := input.NumberOfImages
    n <= MAX_NUMBER
}

number_message[msg] {
	number_less
	msg = sprintf("there's <= %d images", [MAX_NUMBER])
}

number_message[msg] {
	not number_less
	msg = sprintf("there's > %d images", [MAX_NUMBER])
}

size_less {
    s := input.SizeOfImages
    s <= MAX_SIZE
}

size_message[msg] {
	size_less
	msg = sprintf("images take <= %d B", [MAX_SIZE])
}

size_message[msg] {
	not size_less
	msg = sprintf("images take > %d B", [MAX_SIZE])
}

message[msg] {
	some nm, sm
    number_message[nm]
    size_message[sm]
    msg := messagef([nm, sm])
}

messagef([nm, sm]) = nm {
	not number_less
} else = sm {
	not size_less
}

CREATE TABLE image (id serial PRIMARY KEY, name VARCHAR, fullpath VARCHAR, thumbnailpath VARCHAR, resolution_x INT, resolution_y INT);
CREATE TABLE "user" (id serial PRIMARY KEY, username VARCHAR, password VARCHAR);
INSERT INTO "user" values (1, 'bojan', '49c765a9dc9c3a3fc40ec8afed40167d3d9cf0d5');
INSERT INTO "user" values (2, 'bojan2', '49c765a9dc9c3a3fc40ec8afed40167d3d9cf0d5');
CREATE TABLE user_images (user_id INT NOT NULL, image_id INT NOT NULL, PRIMARY KEY (user_id, image_id), FOREIGN KEY (user_id) REFERENCES "user"(id), FOREIGN KEY (image_id) REFERENCES image(id));
-- +goose Up
insert into video (id, code)
values (1, '6u28g47nlPQ');
       -- (2, 'qjxxYoL7nSU'),
       -- (3, 'hLQl3WQQoQ0'),
       -- (4, 'DeumyOzKqgI'),
       -- (5, 'dQw4w9WgXcQ');
SELECT setval('video_id_seq', (select count(*) from video), true);

insert into stream (id, name, is_active,type, settings)
values (1,'anime',true,'playlist','{"tg_channel_id":-1002410966301,"url":"rtmps://dc4-1.rtmp.t.me/s/2520125714:73vbjhh9n6PDoeNF-S8rFQ","playlist_code":"PLjNlQ2vXx1xbt30X8TcUfNzw_akVISXEu", "resolution":"640x360", "audio_bitrate":44100}'),
       (2,'main2',false,'','{"tg_channel_id":-1001878488754,"url":"rtmps://dc4-1.rtmp.t.me/s/1878488754:mYbLoCqewRZozRCjJyZcew"}');

SELECT setval('stream_id_seq', (select count(*) from stream), true);

insert into playlist_item (id, stream_id, video_id)
values (1,1,1);
       -- (2,1,2),
       -- (3,1,3),
       -- (4,1,4),
       -- (5,2,1),
       -- (6,2,2),
       -- (7,1,5);
SELECT setval('playlist_item_id_seq', (select count(*) from playlist_item), true);



-- insert into api_key (id, stream_id, hash)
-- values (1,1,'tsLoRcz5S8qme5xMrwf5CfsQzAkR2u97a8PzAb6k5XE'),
--        (2,2,'PfXgOOo3hNwIBOM5Ej+Eo1xsaKCx7adbDGIYmY1khR8');
-- SELECT setval('api_key_id_seq', (select count(*) from api_key), true);

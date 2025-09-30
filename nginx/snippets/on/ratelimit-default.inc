limit_req zone=default_limit burst=20 nodelay;
limit_req_status 429;



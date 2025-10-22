limit_req zone=assets_limit burst=50 nodelay;
limit_req_status 429;



limit_req zone=auth_limit burst=10 nodelay;
limit_req_status 429;



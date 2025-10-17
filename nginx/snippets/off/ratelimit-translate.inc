limit_req zone=translate_limit burst=10 nodelay;
limit_req_status 429;

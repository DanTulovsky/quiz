limit_req zone=translate_limit burst=2 nodelay;
limit_req_status 429;

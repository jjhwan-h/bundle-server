SELECT DISTINCT u.gtype, u.gcode
FROM casb.t_policy_saas_user_mapping m
JOIN common.t_profile_user_sub u ON m.pid = u.pid
WHERE m.rule_id = ?

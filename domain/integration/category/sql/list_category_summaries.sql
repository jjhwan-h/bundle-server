		SELECT t1.cid, t1.pid, t1.cname,
CASE 
     WHEN EXISTS (
     		SELECT 1
     		  FROM common.t_saas_url_rw t2
     		 WHERE t1.cid = t2.cid
     ) THEN (
     		SELECT
     			COALESCE(
      			(SELECT  t3.action
     		  		  FROM common.t_profile_saas_cate_sub t3
          		 WHERE t1.cid = t3.cid LIMIT 1
          	),
				(SELECT t4.ACTION FROM casb.t_policy_saas_config t4)
			)
	  )
     ELSE 1
 END AS action
FROM common.t_saas_category t1;
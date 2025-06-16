SELECT t1.cid, 
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
                   WHERE t1.cid = t3.cid and `action` > 1 LIMIT 1
              ),
                (SELECT t4.ACTION FROM casb.t_policy_saas_config t4)
            )
      )
     ELSE 1
 END AS action
FROM (WITH RECURSIVE CTE AS ( 
                SELECT DISTINCT 1 AS idx, cid,  pid  FROM common.t_saas_category 
                WHERE cid IN (SELECT cid FROM common.t_profile_saas_cate_sub WHERE pid IN (?) AND `action` = '1')
                UNION ALL 
                SELECT idx+1, P.cid, P.pid FROM common.t_saas_category P INNER JOIN CTE ON P.pid = CTE.cid AND CTE.idx < 20 
            ) 
        SELECT cid FROM CTE) t1;
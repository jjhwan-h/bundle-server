WITH RECURSIVE CTE AS ( 
				SELECT DISTINCT 1 AS idx, gid,  pid  FROM common.t_org_group 
				WHERE gid IN (?)
				UNION ALL 
				SELECT idx+1, P.gid, P.pid FROM common.t_org_group P INNER JOIN CTE ON P.pid = CTE.gid AND CTE.idx < 20 
			) 
		SELECT gid FROM CTE;
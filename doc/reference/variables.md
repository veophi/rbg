# Labels, Annotations and Environment Variables

## Labels

 Key                                    | Description                                                     
----------------------------------------|-----------------------------------------------------------------
 rolebasedgroup.workloads.x-k8s.io/name | The name of the RoleBasedGroup to which these resources belong. 
 rolebasedgroup.workloads.x-k8s.io/role | The name of the role to which these resources belong.           
 pod-group.scheduling.sigs.k8s.io/name  | The name of the podGroup for gang scheduling.                   

## Annotations

 Key                                         | Description           
---------------------------------------------|-----------------------
 rolebasedgroup.workloads.x-k8s.io/role-size | The size of the role. 

## Env Variables

 Key        | Description                                        
------------|----------------------------------------------------
 GROUP_NAME | The name of the RoleBasedGroup.                    
 ROLE_NAME  | The name of the role.                              
 ROLE_INDEX | The index or identity of the pod within the role.	 


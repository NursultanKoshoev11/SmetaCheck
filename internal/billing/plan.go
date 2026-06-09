package billing

type Plan struct{
 Code string
 MaxFiles int
 MaxFileMB int
 MaxReports int
}

func FreePlan()Plan{return Plan{"free",3,10,3}}
func ProPlan()Plan{return Plan{"pro",100,50,100}}
func CompanyPlan()Plan{return Plan{"company",1000,100,1000}}

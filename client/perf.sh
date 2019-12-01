for i in 1 2 5 8 10 12 15 18 20 22 25 28 30
do
 { time go run main.go direct -d $i -n 1 -block COMSM0010cloud -timeout 1000; } 2>&1 | tee -a results-1w-comsm0010cloud.out
 sleep 60
done

for i in 1 2 5 8 10 12 15 18 20 22 25 28 30
do
 { time go run main.go direct -d $i -n 5 -block COMSM0010cloud -timeout 1000; } 2>&1 | tee -a results2-5w-comsm0010cloud.out
 sleep 60
done

for i in 1 2 5 8 10 12 15 18 20 22 25 28 30
do
 { time go run main.go direct -d $i -n 2 -block COMSM0010cloud -timeout 1000; } 2>&1 | tee -a results-2w-comsm0010cloud.out
 sleep 60
done

for i in 1 2 5 8 10 12 15 18 20 22 25 28 30
do
 { time go run main.go direct -d $i -n 2 -block COMSM0010cloud -timeout 1000; } 2>&1 | tee -a results2-2w-comsm0010cloud.out
 sleep 60
done

for i in 1 2 5 8 10 12 15 18 20 22 25 28 30
do
 { time go run main.go direct -d $i -n 10 -block COMSM0010cloud -timeout 1000; } 2>&1 | tee -a results-10w-comsm0010cloud.out
 sleep 60
done

for i in 1 2 5 8 10 12 15 18 20 22 25 28 30
do
 { time go run main.go direct -d $i -n 10 -block COMSM0010cloud -timeout 1000; } 2>&1 | tee -a results2-10w-comsm0010cloud.out
 sleep 60
done

# test-k8s-controller

我用来测试 k8s 控制器的一个项目。

## 使用

```bash
$ make && ./testbin --kubeconfig ~/.kube/config
go build -o testbin .
I0216 15:07:46.380608 1716371 controller.go:78] "Starting web-app-controller"
I0216 15:07:46.380648 1716371 controller.go:105] "Starting web-app-controller"
I0216 15:07:46.480707 1716371 controller.go:111] "Starting workers"

```

另外一个终端apply一个webapp

```bash
$ kubectl apply -f crd/crd_webapps_jiajunhuang_com.yaml 
customresourcedefinition.apiextensions.k8s.io/webapps.jiajunhuang.com unchanged
$ kubectl apply -f example/webapp.yaml 
webapp.jiajunhuang.com/webapp-sample created
$ kubectl delete webapps webapp-sample
webapp.jiajunhuang.com "webapp-sample" deleted
```

可以看到日志：

```bash
I0216 15:08:31.963311 1716546 task1.go:20] "执行 PreCreate" webapp="webapp-sample" step="step1"
I0216 15:08:31.963325 1716546 task1.go:25] "执行 Create" webapp="webapp-sample" step="step1"
I0216 15:08:31.963330 1716546 task1.go:30] "执行 PostCreate" webapp="webapp-sample" step="step1"
I0216 15:08:31.964083 1716546 task2.go:20] "执行 PreCreate" webapp="webapp-sample" step="step2"
I0216 15:08:31.964092 1716546 task2.go:25] "执行 Create" webapp="webapp-sample" step="step2"
I0216 15:08:31.964097 1716546 task2.go:30] "执行 PostCreate" webapp="webapp-sample" step="step2"
I0216 15:08:31.964868 1716546 task3.go:20] "执行 PreCreate" webapp="webapp-sample" step="step3"
I0216 15:08:31.964875 1716546 task3.go:25] "执行 Create" webapp="webapp-sample" step="step3"
I0216 15:08:31.964879 1716546 task3.go:30] "执行 PostCreate" webapp="webapp-sample" step="step3"
I0216 15:08:31.965583 1716546 task1.go:20] "执行 PreCreate" webapp="webapp-sample" step="step1"
I0216 15:08:31.965590 1716546 task1.go:25] "执行 Create" webapp="webapp-sample" step="step1"
I0216 15:08:31.965594 1716546 task1.go:30] "执行 PostCreate" webapp="webapp-sample" step="step1"
I0216 15:08:31.966385 1716546 task2.go:20] "执行 PreCreate" webapp="webapp-sample" step="step2"
I0216 15:08:31.966394 1716546 task2.go:25] "执行 Create" webapp="webapp-sample" step="step2"
I0216 15:08:31.966400 1716546 task2.go:30] "执行 PostCreate" webapp="webapp-sample" step="step2"
I0216 15:08:31.967157 1716546 task3.go:20] "执行 PreCreate" webapp="webapp-sample" step="step3"
I0216 15:08:31.967165 1716546 task3.go:25] "执行 Create" webapp="webapp-sample" step="step3"
I0216 15:08:31.967169 1716546 task3.go:30] "执行 PostCreate" webapp="webapp-sample" step="step3"
I0216 15:08:34.701611 1716546 task3.go:35] "执行 PreDelete" webapp="webapp-sample" step="step3"
I0216 15:08:34.701622 1716546 task3.go:40] "执行 Delete" webapp="webapp-sample" step="step3"
I0216 15:08:34.701629 1716546 task3.go:45] "执行 PostDelete" webapp="webapp-sample" step="step3"
I0216 15:08:34.701637 1716546 controller.go:215] "成功完成删除操作" webapp="webapp-sample" executor="*tasks.Step3TaskExecutor"
I0216 15:08:34.702551 1716546 task2.go:35] "执行 PreDelete" webapp="webapp-sample" step="step2"
I0216 15:08:34.702561 1716546 task2.go:40] "执行 Delete" webapp="webapp-sample" step="step2"
I0216 15:08:34.702566 1716546 task2.go:45] "执行 PostDelete" webapp="webapp-sample" step="step2"
I0216 15:08:34.702574 1716546 controller.go:215] "成功完成删除操作" webapp="webapp-sample" executor="*tasks.Step2TaskExecutor"
I0216 15:08:34.703391 1716546 task1.go:35] "执行 PreDelete" webapp="webapp-sample" step="step1"
I0216 15:08:34.703401 1716546 task1.go:40] "执行 Delete" webapp="webapp-sample" step="step1"
I0216 15:08:34.703407 1716546 task1.go:45] "执行 PostDelete" webapp="webapp-sample" step="step1"
I0216 15:08:34.706199 1716546 controller.go:215] "成功完成删除操作" webapp="webapp-sample" executor="*tasks.Step1TaskExecutor"
I0216 15:08:34.706223 1716546 controller.go:151] "WebApp 已经被删除" webapp="default/webapp-sample"

```

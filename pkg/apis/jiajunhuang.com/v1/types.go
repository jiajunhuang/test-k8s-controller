package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebApp 是一个自定义资源示例
type WebApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebAppSpec   `json:"spec"`
	Status WebAppStatus `json:"status,omitempty"`
}

// WebAppSpec 定义了 WebApp 的期望状态
type WebAppSpec struct {
	// 在这里添加你的规格字段
	Image    string `json:"image"`
	Version  string `json:"version"`
	Replicas int32  `json:"replicas"`
}

// WebAppStatus 定义了 WebApp 的实际状态
type WebAppStatus struct {
	// 在这里添加你的状态字段
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebAppList 包含 WebApp 的列表
type WebAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []WebApp `json:"items"`
}

# Pipeline CRD 字段说明

Pipeline CRD 是一个用于管理日志采集配置的 Kubernetes 自定义资源，它支持灵活的日志采集、处理和输出配置，并集成了阿里云日志服务（SLS）的功能。

## 顶层字段

| 字段名 | 类型 | 是否必填 | 说明 |
|--------|------|----------|------|
| apiVersion | string | 是 | 固定值：`infraflow.co/v1alpha1` |
| kind | string | 是 | 固定值：`Pipeline` |
| metadata | object | 是 | 元数据信息 |
| spec | object | 是 | Pipeline 的详细配置 |
| status | object | 否 | Pipeline 的运行状态 |

## spec 字段

| 字段名 | 类型 | 是否必填 | 说明 |
|--------|------|----------|------|
| name | string | 是 | Pipeline 的名称 |
| content | string | 是 | Pipeline 的配置内容 |
| agentGroup | string | 否 | 指定应用此 Pipeline 的 Agent 组 |
| project | object | 否 | SLS Project 配置 |
| logStores | object | 否 | SLS Logstore 配置 |
| machineGroups | object | 否 | 日志采集的机器组配置 |
| enableUpgradeOverride | bool | 否 | 是否启用升级覆盖 |

## status 字段

| 字段名 | 类型 | 是否必填 | 说明 |
|--------|------|----------|------|
| success | bool | 是 | Pipeline 是否创建成功 |
| message | string | 否 | Pipeline 的状态信息 |
| lastUpdateTime | string | 否 | Pipeline 最后更新时间 |
| lastAppliedConfig | object | 否 | 最后应用的配置信息 |

### lastAppliedConfig 字段

| 字段名 | 类型 | 是否必填 | 说明 |
|--------|------|----------|------|
| appliedTime | string | 否 | 配置应用时间 |
| content | string | 否 | 应用的配置内容 |

## 使用示例

```yaml
apiVersion: infraflow.co/v1alpha1
kind: Pipeline
metadata:
  name: example-pipeline
spec:
  name: "example-pipeline"
  content: |
    inputs:
      - type: file
        paths:
          - /var/log/*.log
    processors:
      - type: json
    outputs:
      - type: sls
        endpoint: cn-hangzhou.log.aliyuncs.com
        project: example-project
        logstore: example-logstore
  agentGroup: "example-group"
  project:
    name: "example-project"
  logStores:
    - name: "example-logstore"
      ttl: 30
      shardCount: 2
  machineGroups:
    - name: "example-group"
  enableUpgradeOverride: true
```

## 注意事项

1. Pipeline 的 `content` 字段必须包含有效的配置内容
2. 当指定 `agentGroup` 时，确保该组已经存在
3. `project` 和 `logStores` 配置是可选的，但建议在需要 SLS 集成时提供
4. `enableUpgradeOverride` 默认为 false，设置为 true 时允许在升级时覆盖现有配置

## 更多参考

- [使用AliyunPipelineConfig管理采集配置](https://help.aliyun.com/zh/sls/user-guide/recommend-use-aliyunpipelineconfig-to-manage-collection-configurations?spm=a2c4g.11186623.help-menu-28958.d_2_1_1_3_2_0.55a6683fYqbxFu)
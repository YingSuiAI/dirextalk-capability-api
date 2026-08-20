//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

/// Current Message Server-issued owner session scope constants. This list is authoritative and is not client-selectable.
class AgentDataScope {
  /// Instantiate a new enum with the provided [value].
  const AgentDataScope._(this.value);

  /// The underlying value of this enum member.
  final String value;

  @override
  String toString() => value;

  String toJson() => value;

  static const agentPeriodExecutionPeriodV2 = AgentDataScope._(r'agent.execution.v2');
  static const agentColonAccountColonDeprovision = AgentDataScope._(r'agent:account:deprovision');
  static const agentColonAwsColonCredentialsColonRead = AgentDataScope._(r'agent:aws:credentials:read');
  static const agentColonAwsColonCredentialsColonWrite = AgentDataScope._(r'agent:aws:credentials:write');
  static const agentColonChatColonRead = AgentDataScope._(r'agent:chat:read');
  static const agentColonChatColonWrite = AgentDataScope._(r'agent:chat:write');
  static const agentColonConfigColonRead = AgentDataScope._(r'agent:config:read');
  static const agentColonConfigColonWrite = AgentDataScope._(r'agent:config:write');
  static const agentColonConfirmationsColonRead = AgentDataScope._(r'agent:confirmations:read');
  static const agentColonConfirmationsColonWrite = AgentDataScope._(r'agent:confirmations:write');
  static const agentColonImageToolsColonExecute = AgentDataScope._(r'agent:image_tools:execute');
  static const agentColonImageToolsColonUpload = AgentDataScope._(r'agent:image_tools:upload');
  static const agentColonInfoColonRead = AgentDataScope._(r'agent:info:read');
  static const agentColonKnowledgeColonRead = AgentDataScope._(r'agent:knowledge:read');
  static const agentColonKnowledgeColonWrite = AgentDataScope._(r'agent:knowledge:write');
  static const agentColonMcpColonExecute = AgentDataScope._(r'agent:mcp:execute');
  static const agentColonMcpColonRead = AgentDataScope._(r'agent:mcp:read');
  static const agentColonMcpColonWrite = AgentDataScope._(r'agent:mcp:write');
  static const agentColonMemoryColonRead = AgentDataScope._(r'agent:memory:read');
  static const agentColonMemoryColonWrite = AgentDataScope._(r'agent:memory:write');
  static const agentColonModelsColonRead = AgentDataScope._(r'agent:models:read');
  static const agentColonModelsColonWrite = AgentDataScope._(r'agent:models:write');
  static const agentColonProductColonExecute = AgentDataScope._(r'agent:product:execute');
  static const agentColonRuntimeColonRead = AgentDataScope._(r'agent:runtime:read');
  static const agentColonRuntimeColonWrite = AgentDataScope._(r'agent:runtime:write');
  static const agentColonSchedulesColonRead = AgentDataScope._(r'agent:schedules:read');
  static const agentColonSchedulesColonWrite = AgentDataScope._(r'agent:schedules:write');
  static const agentColonServersColonDestroy = AgentDataScope._(r'agent:servers:destroy');
  static const agentColonServersColonRead = AgentDataScope._(r'agent:servers:read');
  static const agentColonServersColonWrite = AgentDataScope._(r'agent:servers:write');
  static const agentColonSkillsColonExecute = AgentDataScope._(r'agent:skills:execute');
  static const agentColonSkillsColonRead = AgentDataScope._(r'agent:skills:read');
  static const agentColonSkillsColonWrite = AgentDataScope._(r'agent:skills:write');
  static const agentColonStaticSitesColonRead = AgentDataScope._(r'agent:static_sites:read');
  static const agentColonStaticSitesColonWrite = AgentDataScope._(r'agent:static_sites:write');
  static const agentColonTasksColonRead = AgentDataScope._(r'agent:tasks:read');
  static const agentColonTasksColonWrite = AgentDataScope._(r'agent:tasks:write');
  static const agentColonTextToolsColonExecute = AgentDataScope._(r'agent:text_tools:execute');
  static const agentColonTextToolsColonRead = AgentDataScope._(r'agent:text_tools:read');
  static const agentColonTextToolsColonWrite = AgentDataScope._(r'agent:text_tools:write');
  static const agentColonVoiceColonWrite = AgentDataScope._(r'agent:voice:write');
  static const agentColonWebSearchColonRead = AgentDataScope._(r'agent:web_search:read');
  static const agentColonWebSearchColonWrite = AgentDataScope._(r'agent:web_search:write');
  static const agentColonWorkerColonDestroy = AgentDataScope._(r'agent:worker:destroy');
  static const agentColonWorkerColonRead = AgentDataScope._(r'agent:worker:read');

  /// List of all possible values in this [enum][AgentDataScope].
  static const values = <AgentDataScope>[
    agentPeriodExecutionPeriodV2,
    agentColonAccountColonDeprovision,
    agentColonAwsColonCredentialsColonRead,
    agentColonAwsColonCredentialsColonWrite,
    agentColonChatColonRead,
    agentColonChatColonWrite,
    agentColonConfigColonRead,
    agentColonConfigColonWrite,
    agentColonConfirmationsColonRead,
    agentColonConfirmationsColonWrite,
    agentColonImageToolsColonExecute,
    agentColonImageToolsColonUpload,
    agentColonInfoColonRead,
    agentColonKnowledgeColonRead,
    agentColonKnowledgeColonWrite,
    agentColonMcpColonExecute,
    agentColonMcpColonRead,
    agentColonMcpColonWrite,
    agentColonMemoryColonRead,
    agentColonMemoryColonWrite,
    agentColonModelsColonRead,
    agentColonModelsColonWrite,
    agentColonProductColonExecute,
    agentColonRuntimeColonRead,
    agentColonRuntimeColonWrite,
    agentColonSchedulesColonRead,
    agentColonSchedulesColonWrite,
    agentColonServersColonDestroy,
    agentColonServersColonRead,
    agentColonServersColonWrite,
    agentColonSkillsColonExecute,
    agentColonSkillsColonRead,
    agentColonSkillsColonWrite,
    agentColonStaticSitesColonRead,
    agentColonStaticSitesColonWrite,
    agentColonTasksColonRead,
    agentColonTasksColonWrite,
    agentColonTextToolsColonExecute,
    agentColonTextToolsColonRead,
    agentColonTextToolsColonWrite,
    agentColonVoiceColonWrite,
    agentColonWebSearchColonRead,
    agentColonWebSearchColonWrite,
    agentColonWorkerColonDestroy,
    agentColonWorkerColonRead,
  ];

  static AgentDataScope? fromJson(dynamic value) => AgentDataScopeTypeTransformer().decode(value);

  static List<AgentDataScope> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <AgentDataScope>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AgentDataScope.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AgentDataScope] to String,
/// and [decode] dynamic data back to [AgentDataScope].
class AgentDataScopeTypeTransformer {
  factory AgentDataScopeTypeTransformer() => _instance ??= const AgentDataScopeTypeTransformer._();

  const AgentDataScopeTypeTransformer._();

  String encode(AgentDataScope data) => data.value;

  /// Decodes a [dynamic value][data] to a AgentDataScope.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AgentDataScope? decode(dynamic data, {bool allowNull = true}) {
    if (data != null) {
      switch (data) {
        case r'agent.execution.v2': return AgentDataScope.agentPeriodExecutionPeriodV2;
        case r'agent:account:deprovision': return AgentDataScope.agentColonAccountColonDeprovision;
        case r'agent:aws:credentials:read': return AgentDataScope.agentColonAwsColonCredentialsColonRead;
        case r'agent:aws:credentials:write': return AgentDataScope.agentColonAwsColonCredentialsColonWrite;
        case r'agent:chat:read': return AgentDataScope.agentColonChatColonRead;
        case r'agent:chat:write': return AgentDataScope.agentColonChatColonWrite;
        case r'agent:config:read': return AgentDataScope.agentColonConfigColonRead;
        case r'agent:config:write': return AgentDataScope.agentColonConfigColonWrite;
        case r'agent:confirmations:read': return AgentDataScope.agentColonConfirmationsColonRead;
        case r'agent:confirmations:write': return AgentDataScope.agentColonConfirmationsColonWrite;
        case r'agent:image_tools:execute': return AgentDataScope.agentColonImageToolsColonExecute;
        case r'agent:image_tools:upload': return AgentDataScope.agentColonImageToolsColonUpload;
        case r'agent:info:read': return AgentDataScope.agentColonInfoColonRead;
        case r'agent:knowledge:read': return AgentDataScope.agentColonKnowledgeColonRead;
        case r'agent:knowledge:write': return AgentDataScope.agentColonKnowledgeColonWrite;
        case r'agent:mcp:execute': return AgentDataScope.agentColonMcpColonExecute;
        case r'agent:mcp:read': return AgentDataScope.agentColonMcpColonRead;
        case r'agent:mcp:write': return AgentDataScope.agentColonMcpColonWrite;
        case r'agent:memory:read': return AgentDataScope.agentColonMemoryColonRead;
        case r'agent:memory:write': return AgentDataScope.agentColonMemoryColonWrite;
        case r'agent:models:read': return AgentDataScope.agentColonModelsColonRead;
        case r'agent:models:write': return AgentDataScope.agentColonModelsColonWrite;
        case r'agent:product:execute': return AgentDataScope.agentColonProductColonExecute;
        case r'agent:runtime:read': return AgentDataScope.agentColonRuntimeColonRead;
        case r'agent:runtime:write': return AgentDataScope.agentColonRuntimeColonWrite;
        case r'agent:schedules:read': return AgentDataScope.agentColonSchedulesColonRead;
        case r'agent:schedules:write': return AgentDataScope.agentColonSchedulesColonWrite;
        case r'agent:servers:destroy': return AgentDataScope.agentColonServersColonDestroy;
        case r'agent:servers:read': return AgentDataScope.agentColonServersColonRead;
        case r'agent:servers:write': return AgentDataScope.agentColonServersColonWrite;
        case r'agent:skills:execute': return AgentDataScope.agentColonSkillsColonExecute;
        case r'agent:skills:read': return AgentDataScope.agentColonSkillsColonRead;
        case r'agent:skills:write': return AgentDataScope.agentColonSkillsColonWrite;
        case r'agent:static_sites:read': return AgentDataScope.agentColonStaticSitesColonRead;
        case r'agent:static_sites:write': return AgentDataScope.agentColonStaticSitesColonWrite;
        case r'agent:tasks:read': return AgentDataScope.agentColonTasksColonRead;
        case r'agent:tasks:write': return AgentDataScope.agentColonTasksColonWrite;
        case r'agent:text_tools:execute': return AgentDataScope.agentColonTextToolsColonExecute;
        case r'agent:text_tools:read': return AgentDataScope.agentColonTextToolsColonRead;
        case r'agent:text_tools:write': return AgentDataScope.agentColonTextToolsColonWrite;
        case r'agent:voice:write': return AgentDataScope.agentColonVoiceColonWrite;
        case r'agent:web_search:read': return AgentDataScope.agentColonWebSearchColonRead;
        case r'agent:web_search:write': return AgentDataScope.agentColonWebSearchColonWrite;
        case r'agent:worker:destroy': return AgentDataScope.agentColonWorkerColonDestroy;
        case r'agent:worker:read': return AgentDataScope.agentColonWorkerColonRead;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// Singleton [AgentDataScopeTypeTransformer] instance.
  static AgentDataScopeTypeTransformer? _instance;
}


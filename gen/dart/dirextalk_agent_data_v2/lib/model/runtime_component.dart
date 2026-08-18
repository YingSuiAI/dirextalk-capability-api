//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;


class RuntimeComponent {
  /// Instantiate a new enum with the provided [value].
  const RuntimeComponent._(this.value);

  /// The underlying value of this enum member.
  final String value;

  @override
  String toString() => value;

  String toJson() => value;

  static const systemPrompt = RuntimeComponent._(r'system_prompt');
  static const requestDialect = RuntimeComponent._(r'request_dialect');
  static const intrinsicTools = RuntimeComponent._(r'intrinsic_tools');
  static const extensions = RuntimeComponent._(r'extensions');
  static const executionPolicy = RuntimeComponent._(r'execution_policy');

  /// List of all possible values in this [enum][RuntimeComponent].
  static const values = <RuntimeComponent>[
    systemPrompt,
    requestDialect,
    intrinsicTools,
    extensions,
    executionPolicy,
  ];

  static RuntimeComponent? fromJson(dynamic value) => RuntimeComponentTypeTransformer().decode(value);

  static List<RuntimeComponent> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <RuntimeComponent>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = RuntimeComponent.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [RuntimeComponent] to String,
/// and [decode] dynamic data back to [RuntimeComponent].
class RuntimeComponentTypeTransformer {
  factory RuntimeComponentTypeTransformer() => _instance ??= const RuntimeComponentTypeTransformer._();

  const RuntimeComponentTypeTransformer._();

  String encode(RuntimeComponent data) => data.value;

  /// Decodes a [dynamic value][data] to a RuntimeComponent.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  RuntimeComponent? decode(dynamic data, {bool allowNull = true}) {
    if (data != null) {
      switch (data) {
        case r'system_prompt': return RuntimeComponent.systemPrompt;
        case r'request_dialect': return RuntimeComponent.requestDialect;
        case r'intrinsic_tools': return RuntimeComponent.intrinsicTools;
        case r'extensions': return RuntimeComponent.extensions;
        case r'execution_policy': return RuntimeComponent.executionPolicy;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// Singleton [RuntimeComponentTypeTransformer] instance.
  static RuntimeComponentTypeTransformer? _instance;
}


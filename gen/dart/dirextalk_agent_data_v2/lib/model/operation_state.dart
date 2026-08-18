//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

/// Generic operation and durable Turn states currently exposed by the data plane.
class OperationState {
  /// Instantiate a new enum with the provided [value].
  const OperationState._(this.value);

  /// The underlying value of this enum member.
  final String value;

  @override
  String toString() => value;

  String toJson() => value;

  static const pending = OperationState._(r'pending');
  static const accepted = OperationState._(r'accepted');
  static const running = OperationState._(r'running');
  static const waitingConfirmation = OperationState._(r'waiting_confirmation');
  static const completed = OperationState._(r'completed');
  static const failed = OperationState._(r'failed');
  static const canceled = OperationState._(r'canceled');
  static const cancelled = OperationState._(r'cancelled');
  static const uncertain = OperationState._(r'uncertain');

  /// List of all possible values in this [enum][OperationState].
  static const values = <OperationState>[
    pending,
    accepted,
    running,
    waitingConfirmation,
    completed,
    failed,
    canceled,
    cancelled,
    uncertain,
  ];

  static OperationState? fromJson(dynamic value) => OperationStateTypeTransformer().decode(value);

  static List<OperationState> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <OperationState>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OperationState.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [OperationState] to String,
/// and [decode] dynamic data back to [OperationState].
class OperationStateTypeTransformer {
  factory OperationStateTypeTransformer() => _instance ??= const OperationStateTypeTransformer._();

  const OperationStateTypeTransformer._();

  String encode(OperationState data) => data.value;

  /// Decodes a [dynamic value][data] to a OperationState.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  OperationState? decode(dynamic data, {bool allowNull = true}) {
    if (data != null) {
      switch (data) {
        case r'pending': return OperationState.pending;
        case r'accepted': return OperationState.accepted;
        case r'running': return OperationState.running;
        case r'waiting_confirmation': return OperationState.waitingConfirmation;
        case r'completed': return OperationState.completed;
        case r'failed': return OperationState.failed;
        case r'canceled': return OperationState.canceled;
        case r'cancelled': return OperationState.cancelled;
        case r'uncertain': return OperationState.uncertain;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// Singleton [OperationStateTypeTransformer] instance.
  static OperationStateTypeTransformer? _instance;
}


//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

class TurnOperationReceipt {
  /// Returns a new [TurnOperationReceipt] instance.
  TurnOperationReceipt({
    required this.operationId,
    required this.turnId,
    required this.idempotencyKey,
    required this.state,
    required this.replayed,
  });

  /// Durable operation UUID.
  String operationId;

  /// Durable Turn UUID.
  String turnId;

  String idempotencyKey;

  OperationState state;

  bool replayed;

  @override
  bool operator ==(Object other) => identical(this, other) || other is TurnOperationReceipt &&
    other.operationId == operationId &&
    other.turnId == turnId &&
    other.idempotencyKey == idempotencyKey &&
    other.state == state &&
    other.replayed == replayed;

  @override
  int get hashCode =>
    // ignore: unnecessary_parenthesis
    (operationId.hashCode) +
    (turnId.hashCode) +
    (idempotencyKey.hashCode) +
    (state.hashCode) +
    (replayed.hashCode);

  @override
  String toString() => 'TurnOperationReceipt[operationId=$operationId, turnId=$turnId, idempotencyKey=$idempotencyKey, state=$state, replayed=$replayed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
      json[r'operation_id'] = this.operationId;
      json[r'turn_id'] = this.turnId;
      json[r'idempotency_key'] = this.idempotencyKey;
      json[r'state'] = this.state;
      json[r'replayed'] = this.replayed;
    return json;
  }

  /// Returns a new [TurnOperationReceipt] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TurnOperationReceipt? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        requiredKeys.forEach((key) {
          assert(json.containsKey(key), 'Required key "TurnOperationReceipt[$key]" is missing from JSON.');
          assert(json[key] != null, 'Required key "TurnOperationReceipt[$key]" has a null value in JSON.');
        });
        return true;
      }());

      return TurnOperationReceipt(
        operationId: mapValueOfType<String>(json, r'operation_id')!,
        turnId: mapValueOfType<String>(json, r'turn_id')!,
        idempotencyKey: mapValueOfType<String>(json, r'idempotency_key')!,
        state: OperationState.fromJson(json[r'state'])!,
        replayed: mapValueOfType<bool>(json, r'replayed')!,
      );
    }
    return null;
  }

  static List<TurnOperationReceipt> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <TurnOperationReceipt>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TurnOperationReceipt.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TurnOperationReceipt> mapFromJson(dynamic json) {
    final map = <String, TurnOperationReceipt>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TurnOperationReceipt.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TurnOperationReceipt-objects as value to a dart map
  static Map<String, List<TurnOperationReceipt>> mapListFromJson(dynamic json, {bool growable = false,}) {
    final map = <String, List<TurnOperationReceipt>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TurnOperationReceipt.listFromJson(entry.value, growable: growable,);
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'operation_id',
    'turn_id',
    'idempotency_key',
    'state',
    'replayed',
  };
}


//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

class OperationSnapshot {
  /// Returns a new [OperationSnapshot] instance.
  OperationSnapshot({
    required this.operationId,
    this.turnId,
    this.conversationId,
    required this.state,
    required this.sequence,
    this.result,
    this.error,
  });

  /// Durable operation UUID.
  String operationId;

  /// Durable Turn UUID.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? turnId;

  /// Durable conversation UUID.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? conversationId;

  OperationState state;

  /// Monotonic per-operation replay cursor.
  ///
  /// Minimum value: 0
  int sequence;

  Object? result;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  ErrorEnvelope? error;

  @override
  bool operator ==(Object other) => identical(this, other) || other is OperationSnapshot &&
    other.operationId == operationId &&
    other.turnId == turnId &&
    other.conversationId == conversationId &&
    other.state == state &&
    other.sequence == sequence &&
    other.result == result &&
    other.error == error;

  @override
  int get hashCode =>
    // ignore: unnecessary_parenthesis
    (operationId.hashCode) +
    (turnId == null ? 0 : turnId!.hashCode) +
    (conversationId == null ? 0 : conversationId!.hashCode) +
    (state.hashCode) +
    (sequence.hashCode) +
    (result == null ? 0 : result!.hashCode) +
    (error == null ? 0 : error!.hashCode);

  @override
  String toString() => 'OperationSnapshot[operationId=$operationId, turnId=$turnId, conversationId=$conversationId, state=$state, sequence=$sequence, result=$result, error=$error]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
      json[r'operation_id'] = this.operationId;
    if (this.turnId != null) {
      json[r'turn_id'] = this.turnId;
    } else {
      json[r'turn_id'] = null;
    }
    if (this.conversationId != null) {
      json[r'conversation_id'] = this.conversationId;
    } else {
      json[r'conversation_id'] = null;
    }
      json[r'state'] = this.state;
      json[r'sequence'] = this.sequence;
    if (this.result != null) {
      json[r'result'] = this.result;
    } else {
      json[r'result'] = null;
    }
    if (this.error != null) {
      json[r'error'] = this.error;
    } else {
      json[r'error'] = null;
    }
    return json;
  }

  /// Returns a new [OperationSnapshot] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static OperationSnapshot? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        requiredKeys.forEach((key) {
          assert(json.containsKey(key), 'Required key "OperationSnapshot[$key]" is missing from JSON.');
          assert(json[key] != null, 'Required key "OperationSnapshot[$key]" has a null value in JSON.');
        });
        return true;
      }());

      return OperationSnapshot(
        operationId: mapValueOfType<String>(json, r'operation_id')!,
        turnId: mapValueOfType<String>(json, r'turn_id'),
        conversationId: mapValueOfType<String>(json, r'conversation_id'),
        state: OperationState.fromJson(json[r'state'])!,
        sequence: mapValueOfType<int>(json, r'sequence')!,
        result: mapValueOfType<Object>(json, r'result'),
        error: ErrorEnvelope.fromJson(json[r'error']),
      );
    }
    return null;
  }

  static List<OperationSnapshot> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <OperationSnapshot>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OperationSnapshot.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, OperationSnapshot> mapFromJson(dynamic json) {
    final map = <String, OperationSnapshot>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = OperationSnapshot.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of OperationSnapshot-objects as value to a dart map
  static Map<String, List<OperationSnapshot>> mapListFromJson(dynamic json, {bool growable = false,}) {
    final map = <String, List<OperationSnapshot>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = OperationSnapshot.listFromJson(entry.value, growable: growable,);
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'operation_id',
    'state',
    'sequence',
  };
}


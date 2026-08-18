//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

class ErrorDetails {
  /// Returns a new [ErrorDetails] instance.
  ErrorDetails({
    this.field,
    this.requiredScope,
    this.expectedRevision,
    this.actualRevision,
    this.firstSequence,
    this.lastSequence,
    this.runtimeComponent,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? field;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  AgentDataScope? requiredScope;

  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? expectedRevision;

  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? actualRevision;

  /// Monotonic per-operation replay cursor.
  ///
  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? firstSequence;

  /// Monotonic per-operation replay cursor.
  ///
  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? lastSequence;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  RuntimeComponent? runtimeComponent;

  @override
  bool operator ==(Object other) => identical(this, other) || other is ErrorDetails &&
    other.field == field &&
    other.requiredScope == requiredScope &&
    other.expectedRevision == expectedRevision &&
    other.actualRevision == actualRevision &&
    other.firstSequence == firstSequence &&
    other.lastSequence == lastSequence &&
    other.runtimeComponent == runtimeComponent;

  @override
  int get hashCode =>
    // ignore: unnecessary_parenthesis
    (field == null ? 0 : field!.hashCode) +
    (requiredScope == null ? 0 : requiredScope!.hashCode) +
    (expectedRevision == null ? 0 : expectedRevision!.hashCode) +
    (actualRevision == null ? 0 : actualRevision!.hashCode) +
    (firstSequence == null ? 0 : firstSequence!.hashCode) +
    (lastSequence == null ? 0 : lastSequence!.hashCode) +
    (runtimeComponent == null ? 0 : runtimeComponent!.hashCode);

  @override
  String toString() => 'ErrorDetails[field=$field, requiredScope=$requiredScope, expectedRevision=$expectedRevision, actualRevision=$actualRevision, firstSequence=$firstSequence, lastSequence=$lastSequence, runtimeComponent=$runtimeComponent]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.field != null) {
      json[r'field'] = this.field;
    } else {
      json[r'field'] = null;
    }
    if (this.requiredScope != null) {
      json[r'required_scope'] = this.requiredScope;
    } else {
      json[r'required_scope'] = null;
    }
    if (this.expectedRevision != null) {
      json[r'expected_revision'] = this.expectedRevision;
    } else {
      json[r'expected_revision'] = null;
    }
    if (this.actualRevision != null) {
      json[r'actual_revision'] = this.actualRevision;
    } else {
      json[r'actual_revision'] = null;
    }
    if (this.firstSequence != null) {
      json[r'first_sequence'] = this.firstSequence;
    } else {
      json[r'first_sequence'] = null;
    }
    if (this.lastSequence != null) {
      json[r'last_sequence'] = this.lastSequence;
    } else {
      json[r'last_sequence'] = null;
    }
    if (this.runtimeComponent != null) {
      json[r'runtime_component'] = this.runtimeComponent;
    } else {
      json[r'runtime_component'] = null;
    }
    return json;
  }

  /// Returns a new [ErrorDetails] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ErrorDetails? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        requiredKeys.forEach((key) {
          assert(json.containsKey(key), 'Required key "ErrorDetails[$key]" is missing from JSON.');
          assert(json[key] != null, 'Required key "ErrorDetails[$key]" has a null value in JSON.');
        });
        return true;
      }());

      return ErrorDetails(
        field: mapValueOfType<String>(json, r'field'),
        requiredScope: AgentDataScope.fromJson(json[r'required_scope']),
        expectedRevision: mapValueOfType<int>(json, r'expected_revision'),
        actualRevision: mapValueOfType<int>(json, r'actual_revision'),
        firstSequence: mapValueOfType<int>(json, r'first_sequence'),
        lastSequence: mapValueOfType<int>(json, r'last_sequence'),
        runtimeComponent: RuntimeComponent.fromJson(json[r'runtime_component']),
      );
    }
    return null;
  }

  static List<ErrorDetails> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <ErrorDetails>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ErrorDetails.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ErrorDetails> mapFromJson(dynamic json) {
    final map = <String, ErrorDetails>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ErrorDetails.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ErrorDetails-objects as value to a dart map
  static Map<String, List<ErrorDetails>> mapListFromJson(dynamic json, {bool growable = false,}) {
    final map = <String, List<ErrorDetails>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ErrorDetails.listFromJson(entry.value, growable: growable,);
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
  };
}


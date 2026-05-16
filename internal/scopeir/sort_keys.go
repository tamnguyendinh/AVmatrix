package scopeir

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func compareScope(left ScopeFact, right ScopeFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(string(left.Kind), string(right.Kind)); value != 0 {
		return value
	}
	return compareString(left.ID, right.ID)
}

func compareDefinition(left DefinitionFact, right DefinitionFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(string(left.Label), string(right.Label)); value != 0 {
		return value
	}
	if value := compareString(left.Name, right.Name); value != 0 {
		return value
	}
	return compareString(left.ID, right.ID)
}

func compareImport(left ImportFact, right ImportFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareString(string(left.Kind), string(right.Kind)); value != 0 {
		return value
	}
	if value := compareString(left.LocalName, right.LocalName); value != 0 {
		return value
	}
	if value := compareString(left.ImportedName, right.ImportedName); value != 0 {
		return value
	}
	if value := compareString(deref(left.TargetRaw), deref(right.TargetRaw)); value != 0 {
		return value
	}
	return compareString(left.ID, right.ID)
}

func compareCall(left CallSiteFact, right CallSiteFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.InScope, right.InScope); value != 0 {
		return value
	}
	return compareString(left.Name, right.Name)
}

func compareAccess(left AccessFact, right AccessFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.InScope, right.InScope); value != 0 {
		return value
	}
	if value := compareString(string(left.Kind), string(right.Kind)); value != 0 {
		return value
	}
	return compareString(left.Name, right.Name)
}

func compareHeritage(left HeritageFact, right HeritageFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.InScope, right.InScope); value != 0 {
		return value
	}
	if value := compareString(string(left.Kind), string(right.Kind)); value != 0 {
		return value
	}
	return compareString(left.Name, right.Name)
}

func compareTypeAnnotation(left TypeAnnotationFact, right TypeAnnotationFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.InScope, right.InScope); value != 0 {
		return value
	}
	if value := compareString(left.Name, right.Name); value != 0 {
		return value
	}
	return compareString(left.Type.RawName, right.Type.RawName)
}

func compareReturnType(left ReturnTypeFact, right ReturnTypeFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.DefID, right.DefID); value != 0 {
		return value
	}
	return compareString(left.Type.RawName, right.Type.RawName)
}

func compareFramework(left FrameworkFact, right FrameworkFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.DefID, right.DefID); value != 0 {
		return value
	}
	if value := compareString(left.Framework, right.Framework); value != 0 {
		return value
	}
	return compareString(left.Reason, right.Reason)
}

func compareDomain(left DomainFact, right DomainFact) int {
	if value := compareString(left.FilePath, right.FilePath); value != 0 {
		return value
	}
	if value := compareRange(left.Range, right.Range); value != 0 {
		return value
	}
	if value := compareString(left.DefID, right.DefID); value != 0 {
		return value
	}
	if value := compareString(left.Domain, right.Domain); value != 0 {
		return value
	}
	return compareString(left.Role, right.Role)
}

func compareBinding(left BindingFact, right BindingFact) int {
	if value := compareString(left.Name, right.Name); value != 0 {
		return value
	}
	if value := compareString(left.DefID, right.DefID); value != 0 {
		return value
	}
	if value := compareString(string(left.Origin), string(right.Origin)); value != 0 {
		return value
	}
	return compareString(left.ViaID, right.ViaID)
}

func compareTypeBinding(left TypeBindingFact, right TypeBindingFact) int {
	if value := compareString(left.Name, right.Name); value != 0 {
		return value
	}
	if value := compareString(left.Type.RawName, right.Type.RawName); value != 0 {
		return value
	}
	if value := compareString(left.Type.DeclaredAtScope, right.Type.DeclaredAtScope); value != 0 {
		return value
	}
	return compareString(string(left.Type.Source), string(right.Type.Source))
}

func compareRange(left Range, right Range) int {
	if value := compareInt(left.StartLine, right.StartLine); value != 0 {
		return value
	}
	if value := compareInt(left.StartCol, right.StartCol); value != 0 {
		return value
	}
	if value := compareInt(left.EndLine, right.EndLine); value != 0 {
		return value
	}
	return compareInt(left.EndCol, right.EndCol)
}

func compareString(left string, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareInt(left int, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

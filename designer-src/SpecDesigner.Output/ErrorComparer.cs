using System;
using System.Collections.Generic;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Output
{
	// Token: 0x02000006 RID: 6
	public class ErrorComparer : IEqualityComparer<DocumentErrorsEventArgs>
	{
		// Token: 0x06000019 RID: 25 RVA: 0x00002660 File Offset: 0x00000860
		public bool Equals(DocumentErrorsEventArgs x, DocumentErrorsEventArgs y)
		{
			return x.Description == y.Description && x.ErrorType == y.ErrorType && x.Key == y.Key && x.ProgramKey == y.ProgramKey && x.SourceType == y.SourceType;
		}

		// Token: 0x0600001A RID: 26 RVA: 0x000026C5 File Offset: 0x000008C5
		public int GetHashCode(DocumentErrorsEventArgs obj)
		{
			return obj.GetHashCode();
		}
	}
}

using System;
using System.Collections.Generic;

namespace SpecDesigner
{
	// Token: 0x02000037 RID: 55
	public class CodeTemplateComparer : IEqualityComparer<CodeTemplate>
	{
		// Token: 0x060002B6 RID: 694 RVA: 0x0000C9E4 File Offset: 0x0000ABE4
		public bool Equals(CodeTemplate x, CodeTemplate y)
		{
			return x.Code == y.Code && x.Text == y.Text;
		}

		// Token: 0x060002B7 RID: 695 RVA: 0x0000CA10 File Offset: 0x0000AC10
		public int GetHashCode(CodeTemplate obj)
		{
			return obj.Code.GetHashCode() ^ obj.Text.GetHashCode();
		}
	}
}

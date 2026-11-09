using System;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000046 RID: 70
	internal class ComponentNotFoundException : Exception
	{
		// Token: 0x06000297 RID: 663 RVA: 0x0000DB48 File Offset: 0x0000BD48
		public ComponentNotFoundException(string p)
			: base(p)
		{
		}
	}
}

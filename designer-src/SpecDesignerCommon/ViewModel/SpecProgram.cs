using System;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000014 RID: 20
	public abstract class SpecProgram : AbstractSpecNode
	{
		// Token: 0x060000A5 RID: 165 RVA: 0x00004407 File Offset: 0x00002607
		public SpecProgram(PackageKey key, XElement source)
			: base(key, source)
		{
		}
	}
}

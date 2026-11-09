using System;
using System.Xml.Serialization;

namespace SpecDesignerCommon
{
	// Token: 0x0200002B RID: 43
	public class ObjectBindings
	{
		// Token: 0x04000087 RID: 135
		[XmlArray("TBindings")]
		public TBinding[] bindings;
	}
}

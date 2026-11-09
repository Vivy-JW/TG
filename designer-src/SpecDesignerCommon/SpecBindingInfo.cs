using System;
using System.Xml.Serialization;

namespace SpecDesignerCommon
{
	// Token: 0x0200002A RID: 42
	[XmlRoot("SpecBindingInfo", IsNullable = false, Namespace = "www.digiwin.com")]
	public class SpecBindingInfo
	{
		// Token: 0x04000086 RID: 134
		public ObjectBindings ObjectBindings;
	}
}

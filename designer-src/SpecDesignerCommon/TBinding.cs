using System;
using System.Xml.Serialization;

namespace SpecDesignerCommon
{
	// Token: 0x0200002C RID: 44
	public class TBinding
	{
		// Token: 0x0600015D RID: 349 RVA: 0x00007324 File Offset: 0x00005524
		public override bool Equals(object obj)
		{
			if (!(obj is TBinding))
			{
				return false;
			}
			TBinding tbinding = (TBinding)obj;
			return (this.ObjectName1.Type == tbinding.ObjectName1.Type && this.ObjectName1.Text == tbinding.ObjectName1.Text && this.ObjectName2.Type == tbinding.ObjectName2.Type && this.ObjectName2.Text == tbinding.ObjectName2.Text) || (this.ObjectName1.Type == tbinding.ObjectName2.Type && this.ObjectName1.Text == tbinding.ObjectName2.Text && this.ObjectName2.Type == tbinding.ObjectName1.Type && this.ObjectName2.Text == tbinding.ObjectName1.Text);
		}

		// Token: 0x0600015E RID: 350 RVA: 0x00007418 File Offset: 0x00005618
		public override int GetHashCode()
		{
			return base.GetHashCode();
		}

		// Token: 0x04000088 RID: 136
		[XmlAttribute]
		public int ID;

		// Token: 0x04000089 RID: 137
		[XmlElement(IsNullable = false)]
		public FJSObject ObjectName1;

		// Token: 0x0400008A RID: 138
		[XmlElement(IsNullable = false)]
		public FJSObject ObjectName2;
	}
}

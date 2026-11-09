using System;
using System.Windows;
using System.Windows.Media;

namespace SpecDesigner.FormEditor.Petzold.Media2D
{
	// Token: 0x0200001E RID: 30
	public class ArrowPolyline : ArrowLineBase
	{
		// Token: 0x17000036 RID: 54
		// (get) Token: 0x060000F8 RID: 248 RVA: 0x000059EF File Offset: 0x00003BEF
		// (set) Token: 0x060000F7 RID: 247 RVA: 0x000059E1 File Offset: 0x00003BE1
		public PointCollection Points
		{
			get
			{
				return (PointCollection)base.GetValue(ArrowPolyline.PointsProperty);
			}
			set
			{
				base.SetValue(ArrowPolyline.PointsProperty, value);
			}
		}

		// Token: 0x060000F9 RID: 249 RVA: 0x00005A01 File Offset: 0x00003C01
		public ArrowPolyline()
		{
			this.Points = new PointCollection();
		}

		// Token: 0x17000037 RID: 55
		// (get) Token: 0x060000FA RID: 250 RVA: 0x00005A14 File Offset: 0x00003C14
		protected override Geometry DefiningGeometry
		{
			get
			{
				this.pathgeo.Figures.Clear();
				if (this.Points.Count > 0)
				{
					this.pathfigLine.StartPoint = this.Points[0];
					this.polysegLine.Points.Clear();
					for (int i = 1; i < this.Points.Count; i++)
					{
						this.polysegLine.Points.Add(this.Points[i]);
					}
					this.pathgeo.Figures.Add(this.pathfigLine);
				}
				return base.DefiningGeometry;
			}
		}

		// Token: 0x04000093 RID: 147
		public static readonly DependencyProperty PointsProperty = DependencyProperty.Register("Points", typeof(PointCollection), typeof(ArrowPolyline), new FrameworkPropertyMetadata(null, FrameworkPropertyMetadataOptions.AffectsMeasure));
	}
}
